#!/usr/bin/env bash
# 在本地仓库用 gen-k8s-bundle 与 OpsFleet 数学等价地打 zip，再拷到控制机跑 install.sh（可重试直到成功或次数用尽）。
# 典型两节点：一台 master、一台 worker；由控制机对 inventory 内所有节点 root 免密。
#
# 环境变量（均有默认值，可按需覆盖）：
#   K8S_LAB_SSH        跑 install 的控制机，须能 SSH 到各节点（默认 root@192.168.56.101）
#   K8S_LAB_MASTERS    逗号分隔 master IP（默认 192.168.56.101）
#   K8S_LAB_WORKERS    逗号分隔 worker IP（默认 192.168.56.102）
#   K8S_LAB_CLUSTER    集群名（默认 111111）
#   K8S_LAB_VERSION    如 v1.28.15
#   K8S_LAB_ARCH       amd64|arm64，须与各节点 uname -m 一致（默认 arm64，与 gen-k8s-bundle 一致；x86 用 amd64）
#   K8S_LAB_IMAGESRC   镜像源：aliyun|default|tencent|custom
#   K8S_LAB_OUT        本地 zip 输出路径
#   K8S_LAB_RETRIES    install 失败时最多尝试次数（默认 5）
#   K8S_LAB_RETRY_SLEEP 每次重试前睡眠秒数（默认 60）
#   OPSFLEET_ANSIBLE_DIR  未设置时自动为 <repo>/ansible-agent

set -euo pipefail
REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
export OPSFLEET_ANSIBLE_DIR="${OPSFLEET_ANSIBLE_DIR:-$REPO_ROOT/ansible-agent}"

: "${K8S_LAB_SSH:=root@192.168.56.101}"
: "${K8S_LAB_MASTERS:=192.168.56.101}"
: "${K8S_LAB_WORKERS:=192.168.56.102}"
: "${K8S_LAB_CLUSTER:=111111}"
: "${K8S_LAB_VERSION:=v1.28.15}"
# 与 gen-k8s-bundle 默认一致；x86 实验室可 export K8S_LAB_ARCH=amd64
: "${K8S_LAB_ARCH:=arm64}"
: "${K8S_LAB_IMAGESRC:=aliyun}"
: "${K8S_LAB_OUT:=/tmp/opsfleet-k8s-lab.zip}"
: "${K8S_LAB_RETRIES:=5}"
: "${K8S_LAB_RETRY_SLEEP:=60}"
RDIR="/root/opsfleet-k8s-lab"
LOG_REMOTE="/root/opsfleet-k8s-lab-install.log"

echo "== gen-k8s-bundle (同 UI/HTTP，OPSFLEET_ANSIBLE_DIR=$OPSFLEET_ANSIBLE_DIR) =="
cd "$REPO_ROOT/ft-backend"
go run ./cmd/gen-k8s-bundle \
  -o "$K8S_LAB_OUT" \
  -cluster "$K8S_LAB_CLUSTER" \
  -version "$K8S_LAB_VERSION" \
  -master "$K8S_LAB_MASTERS" \
  -worker "$K8S_LAB_WORKERS" \
  -arch "$K8S_LAB_ARCH" \
  -imageSource "$K8S_LAB_IMAGESRC"

REMOTE_ZIP="/tmp/opsfleet-k8s-lab-$$.zip"
RQ="$(printf %q "$REMOTE_ZIP")"
RDIRQ="$(printf %q "$RDIR")"
LOGQ="$(printf %q "$LOG_REMOTE")"
echo "== scp -> $K8S_LAB_SSH:$REMOTE_ZIP =="
scp -o BatchMode=yes -o ConnectTimeout=20 -o ServerAliveInterval=10 \
  "$K8S_LAB_OUT" "$K8S_LAB_SSH:$REMOTE_ZIP"

attempt=0
ok=0
while [ "$attempt" -lt "$K8S_LAB_RETRIES" ]; do
  attempt=$((attempt + 1))
  echo "== install attempt $attempt / $K8S_LAB_RETRIES =="
  if ssh -o BatchMode=yes -o ConnectTimeout=30 -o ServerAliveInterval=10 \
    "$K8S_LAB_SSH" \
    "export REMOTE_ZIP=$RQ; export RDIR=$RDIRQ; export LOG_REMOTE=$LOGQ; bash -s" <<'INSTALL'
set -euo pipefail
set -o pipefail
rm -rf "$RDIR"
mkdir -p "$RDIR"
cd "$RDIR"
unzip -o -q "$REMOTE_ZIP"
test -f install.sh
# 每次叠加日志，便于对比多次重试
echo "---- run $(date -u +%Y-%m-%dT%H:%M:%SZ) ----" >> "$LOG_REMOTE"
sudo env DEBIAN_FRONTEND=noninteractive bash -x install.sh 2>&1 | tee -a "$LOG_REMOTE"
INSTALL
  then
    ok=1
    break
  fi
  echo "install failed, sleeping ${K8S_LAB_RETRY_SLEEP}s before next attempt (see $K8S_LAB_SSH:$LOG_REMOTE)"
  if [ "$attempt" -lt "$K8S_LAB_RETRIES" ]; then
    sleep "$K8S_LAB_RETRY_SLEEP"
  fi
done

if [ "$ok" -eq 1 ]; then
  echo "== SUCCESS. Log on control: $K8S_LAB_SSH:$LOG_REMOTE =="
  echo "  ssh $K8S_LAB_SSH 'tail -200 $LOG_REMOTE'"
  exit 0
fi
echo "== FAILED after $K8S_LAB_RETRIES attempt(s) =="
exit 1
