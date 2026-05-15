#!/usr/bin/env bash
# 在 K8s 实验机（默认 root@192.168.56.101）上逐个注入异常 Pod，运行 ai-sre diagnose，记录结果后删除。
# 用法：SSH_TARGET=root@192.168.56.101 ./scripts/k8s-go-diagnose-lab-matrix.sh
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
SSH_TARGET="${SSH_TARGET:-root@192.168.56.101}"
NS="${LAB_NS:-aisre-go-lab}"
IMAGE="${LAB_GO_IMAGE:-docker.io/library/memleak-demo:lab}"
RESULT_DIR="${RESULT_DIR:-$ROOT/data/k8s-go-diagnose-lab-$(date +%Y%m%d-%H%M%S)}"
mkdir -p "$RESULT_DIR"

remote() { ssh -o ConnectTimeout=10 "$SSH_TARGET" "$@"; }

echo "==> target=$SSH_TARGET namespace=$NS image=$IMAGE"
echo "==> results -> $RESULT_DIR"

remote "NS='$NS' IMAGE='$IMAGE' bash -s" <<'REMOTE_BOOT'
set -euo pipefail
kubectl create namespace "$NS" --dry-run=client -o yaml | kubectl apply -f -
kubectl label namespace "$NS" aisre-lab=true --overwrite 2>/dev/null || true
# 将 Go 实验镜像同步到 worker（镜像仅 master 导入时 Pod 调度到 worker 会 ImagePullBackOff）
if command -v docker >/dev/null && ssh -o StrictHostKeyChecking=no -o ConnectTimeout=5 root@192.168.56.102 true 2>/dev/null; then
  docker save "$IMAGE" 2>/dev/null | ssh -o StrictHostKeyChecking=no root@192.168.56.102 "ctr -n k8s.io images import -" 2>/dev/null || true
fi
REMOTE_BOOT

remote "NS='$NS' IMAGE='$IMAGE' RESULT_DIR='/tmp/aisre-go-lab-run' bash -s" <<'REMOTE_RUN'
set -euo pipefail
NS="${NS:?}"
IMAGE="${IMAGE:?}"
RESULT_DIR="${RESULT_DIR:?}"
mkdir -p "$RESULT_DIR"
SUMMARY="$RESULT_DIR/summary.tsv"
echo -e "id\tphase\texpect_keywords\texit\troot_cause_snip" > "$SUMMARY"

wait_pod_phase() {
  local name="$1" want="$2" timeout="${3:-120}"
  local i=0
  while [[ $i -lt $timeout ]]; do
    phase=$(kubectl -n "$NS" get pod "$name" -o jsonpath='{.status.phase}' 2>/dev/null || echo Missing)
    reason=$(kubectl -n "$NS" get pod "$name" -o jsonpath='{.status.containerStatuses[0].state.waiting.reason}' 2>/dev/null || true)
    if [[ "$want" == "ImagePullBackOff" && "$reason" == "ImagePullBackOff" ]]; then return 0; fi
    if [[ "$want" == "CrashLoopBackOff" && "$reason" == "CrashLoopBackOff" ]]; then return 0; fi
    if [[ "$want" == "Pending" && "$phase" == "Pending" ]]; then return 0; fi
    if [[ "$want" == "Running" && "$phase" == "Running" ]]; then
      ready=$(kubectl -n "$NS" get pod "$name" -o jsonpath='{.status.conditions[?(@.type=="Ready")].status}' 2>/dev/null || echo False)
      [[ "$ready" == "True" ]] && return 0
    fi
    if [[ "$phase" == "$want" ]]; then return 0; fi
    sleep 2
    i=$((i+2))
  done
  return 1
}

run_case() {
  local id="$1" expect="$2"
  shift 2
  local manifest
  manifest=$(cat)
  echo ""
  echo "======== CASE $id (expect: $expect) ========"
  kubectl -n "$NS" delete pod,deploy -l "aisre-case=$id" --ignore-not-found --wait=false 2>/dev/null || true
  sleep 2
  echo "$manifest" | kubectl -n "$NS" apply -f -
  local pod=""
  for _ in $(seq 1 60); do
    pod=$(kubectl -n "$NS" get pods -l "aisre-case=$id" -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || true)
    [[ -n "$pod" ]] && break
    sleep 2
  done
  if [[ -z "$pod" ]]; then
    echo "$id" "NO_POD" "$expect" "1" "no pod created" >> "$SUMMARY"
    kubectl -n "$NS" delete pod,deploy -l "aisre-case=$id" --ignore-not-found --wait=true --timeout=90s 2>/dev/null || true
    return 1
  fi
  wait_pod_phase "$pod" "$expect" 180 || true
  phase=$(kubectl -n "$NS" get pod "$pod" -o jsonpath='{.status.phase}' 2>/dev/null || echo "?")
  reason=$(kubectl -n "$NS" get pod "$pod" -o jsonpath='{.status.containerStatuses[0].state.waiting.reason}' 2>/dev/null || true)
  echo "pod=$pod phase=$phase reason=$reason"
  out="$RESULT_DIR/${id}.txt"
  set +e
  ai-sre diagnose --pod "$NS/$pod" >"$out" 2>&1
  ec=$?
  set -e
  rc=$(grep -m1 '^根因:' "$out" 2>/dev/null | head -c 200 || echo "(no 根因 line)")
  echo "$id" "$phase/$reason" "$expect" "$ec" "$rc" >> "$SUMMARY"
  kubectl -n "$NS" delete pod,deploy -l "aisre-case=$id" --ignore-not-found --wait=true --timeout=120s 2>/dev/null || true
  sleep 3
}

# 1 ImagePullBackOff
run_case "01_image_pull" "ImagePullBackOff" <<EOF
apiVersion: v1
kind: Pod
metadata:
  name: go-01-image-pull
  labels: {aisre-case: "01_image_pull", app: go-lab}
spec:
  restartPolicy: Never
  containers:
  - name: app
    image: memleak-demo:does-not-exist-tag
    imagePullPolicy: Always
EOF

# 2 CrashLoop exit 1 (Go 容器立即退出)
run_case "02_crash_exit" "CrashLoopBackOff" <<EOF
apiVersion: v1
kind: Pod
metadata:
  name: go-02-crash-exit
  labels: {aisre-case: "02_crash_exit", app: go-lab}
spec:
  containers:
  - name: app
    image: $IMAGE
    imagePullPolicy: IfNotPresent
    command: ["/no-such-go-binary"]
EOF

# 3 OOMKilled (Go memleak + 低 limit)
run_case "03_oom" "CrashLoopBackOff" <<EOF
apiVersion: v1
kind: Pod
metadata:
  name: go-03-oom
  labels: {aisre-case: "03_oom", app: go-lab}
spec:
  containers:
  - name: app
    image: $IMAGE
    imagePullPolicy: IfNotPresent
    env:
    - {name: LEAK_CHUNK_MB, value: "20"}
    - {name: LEAK_INTERVAL_SEC, value: "1"}
    resources:
      limits: {memory: 64Mi}
      requests: {memory: 32Mi}
EOF

# 4 Pending (资源请求过大)
run_case "04_pending" "Pending" <<EOF
apiVersion: v1
kind: Pod
metadata:
  name: go-04-pending
  labels: {aisre-case: "04_pending", app: go-lab}
spec:
  containers:
  - name: app
    image: $IMAGE
    imagePullPolicy: IfNotPresent
    resources:
      requests: {cpu: "64", memory: 128Mi}
EOF

# 5 Liveness 探针失败
run_case "05_liveness" "CrashLoopBackOff" <<EOF
apiVersion: v1
kind: Pod
metadata:
  name: go-05-liveness
  labels: {aisre-case: "05_liveness", app: go-lab}
spec:
  containers:
  - name: app
    image: $IMAGE
    imagePullPolicy: IfNotPresent
    ports: [{containerPort: 8080}]
    livenessProbe:
      httpGet: {path: /health, port: 9090}
      initialDelaySeconds: 3
      periodSeconds: 5
      failureThreshold: 2
EOF

# 6 Readiness 探针失败（Running 但 NotReady）
run_case "06_readiness" "Running" <<EOF
apiVersion: v1
kind: Pod
metadata:
  name: go-06-readiness
  labels: {aisre-case: "06_readiness", app: go-lab}
spec:
  containers:
  - name: app
    image: $IMAGE
    imagePullPolicy: IfNotPresent
    ports: [{containerPort: 8080}]
    readinessProbe:
      httpGet: {path: /health, port: 9090}
      periodSeconds: 3
      failureThreshold: 1
EOF

# 7 命令不存在
run_case "07_bad_cmd" "CrashLoopBackOff" <<EOF
apiVersion: v1
kind: Pod
metadata:
  name: go-07-bad-cmd
  labels: {aisre-case: "07_bad_cmd", app: go-lab}
spec:
  containers:
  - name: app
    image: $IMAGE
    imagePullPolicy: IfNotPresent
    command: ["/usr/local/bin/not-a-go-binary"]
EOF

# 8 Init 容器失败
run_case "08_init_fail" "Pending" <<EOF
apiVersion: v1
kind: Pod
metadata:
  name: go-08-init
  labels: {aisre-case: "08_init_fail", app: go-lab}
spec:
  initContainers:
  - name: init
    image: $IMAGE
    imagePullPolicy: IfNotPresent
    command: ["/bin/sh", "-c", "exit 2"]
  containers:
  - name: app
    image: $IMAGE
    imagePullPolicy: IfNotPresent
EOF

# 9 imagePullPolicy Never 且本地无镜像
run_case "09_never_pull" "Pending" <<EOF
apiVersion: v1
kind: Pod
metadata:
  name: go-09-never
  labels: {aisre-case: "09_never_pull", app: go-lab}
spec:
  containers:
  - name: app
    image: golang:1.22-alpine-not-local
    imagePullPolicy: Never
    command: ["go", "version"]
EOF

# 10 正常 Running（对照组，应识别为内存泄漏风险或 OK）
run_case "10_running_ok" "Running" <<EOF
apiVersion: v1
kind: Pod
metadata:
  name: go-10-running
  labels: {aisre-case: "10_running_ok", app: go-lab}
spec:
  containers:
  - name: app
    image: $IMAGE
    imagePullPolicy: IfNotPresent
    env:
    - {name: LEAK_CHUNK_MB, value: "2"}
    - {name: LEAK_INTERVAL_SEC, value: "5"}
    resources:
      limits: {memory: 256Mi}
EOF

# 11 ErrImagePull (非法仓库)
run_case "11_err_pull" "ImagePullBackOff" <<EOF
apiVersion: v1
kind: Pod
metadata:
  name: go-11-err-pull
  labels: {aisre-case: "11_err_pull", app: go-lab}
spec:
  restartPolicy: Never
  containers:
  - name: app
    image: invalid.registry.local/no-go-app:v1
EOF

# 12 Deployment 级诊断（--deployment）
echo "======== CASE 12_deploy_diag ========"
kubectl -n "$NS" delete deploy -l aisre-case=12_deploy_diag --ignore-not-found --wait=true --timeout=90s 2>/dev/null || true
kubectl -n "$NS" apply -f - <<EOF
apiVersion: apps/v1
kind: Deployment
metadata:
  name: go-12-deploy
  labels: {aisre-case: "12_deploy_diag", app: go-lab}
spec:
  replicas: 1
  selector: {matchLabels: {aisre-case: "12_deploy_diag"}}
  template:
    metadata:
      labels: {aisre-case: "12_deploy_diag", app: go-lab}
    spec:
      containers:
      - name: app
        image: $IMAGE
        imagePullPolicy: IfNotPresent
        env:
        - {name: LEAK_CHUNK_MB, value: "15"}
        - {name: LEAK_INTERVAL_SEC, value: "1"}
        resources:
          limits: {memory: 80Mi}
EOF
sleep 15
pod=$(kubectl -n "$NS" get pods -l aisre-case=12_deploy_diag -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || true)
[[ -n "$pod" ]] && wait_pod_phase "$pod" "CrashLoopBackOff" 120 || true
out="$RESULT_DIR/12_deploy_diag.txt"
set +e
ai-sre diagnose --deployment "$NS/go-12-deploy" >"$out" 2>&1
ec=$?
set -e
rc=$(grep -m1 '^根因:' "$out" 2>/dev/null | head -c 200 || echo "(no 根因)")
echo "12_deploy_diag" "deployment" "OOM/CrashLoop" "$ec" "$rc" >> "$SUMMARY"
kubectl -n "$NS" delete deploy -l aisre-case=12_deploy_diag --ignore-not-found --wait=true --timeout=120s 2>/dev/null || true
sleep 3

echo ""
echo "==> SUMMARY"
cat "$SUMMARY"
REMOTE_RUN

echo "==> fetch results"
scp -r "$SSH_TARGET:/tmp/aisre-go-lab-run/"* "$RESULT_DIR/" 2>/dev/null || remote "tar czf - -C /tmp/aisre-go-lab-run ." | tar xzf - -C "$RESULT_DIR"
echo "Results in $RESULT_DIR"
cat "$RESULT_DIR/summary.tsv" 2>/dev/null || true
