#!/usr/bin/env bash
# 在跑过 install.sh 的「控制机」上，将环境恢复到接近测试前（实验室/回归用）。
# 默认：停常见 systemd 单元、删解压目录与 .opsfleet-k8s-state、保留 /var/cache/opsfleet-k8s。
# --purge-cache   清空二进制缓存目录
# --deep          额外删 /etc/kubernetes、/var/lib/etcd、部分 /usr/local/bin 二进制（破坏性，谨慎）
# 用法：sudo bash scripts/k8s-offline-test-cleanup.sh [/root/某解压目录 ...]

set -euo pipefail

if [[ "${EUID:-0}" -ne 0 ]]; then
  echo "请使用 root 或 sudo 执行"
  exit 1
fi

PURGE_CACHE=0
DEEP=0
DIRS=()

while [[ $# -gt 0 ]]; do
  case "$1" in
    --purge-cache) PURGE_CACHE=1; shift ;;
    --deep)        DEEP=1; shift ;;
    -h|--help)
      sed -n '1,12p' "$0"
      exit 0
      ;;
    *) DIRS+=("$1"); shift ;;
  esac
done

if [[ ${#DIRS[@]} -eq 0 ]]; then
  for d in /root/k8s-unpack /root/arm64-k8s-unpack /root/skill-k8s-unpack /root/opsfleet-k8s-unpack; do
    [[ -d "$d" ]] && DIRS+=("$d")
  done
fi

echo "=== 停止常见测试服务（未安装则忽略）==="
for svc in kube-controller-manager kube-scheduler kube-apiserver etcd; do
  systemctl stop "$svc" 2>/dev/null || true
  systemctl disable "$svc" 2>/dev/null || true
done
systemctl daemon-reload 2>/dev/null || true

for d in "${DIRS[@]}"; do
  echo "=== 删除解压目录: $d ==="
  rm -rf "$d"
done

# 状态文件与解压目录同层或根下残留
shopt -s nullglob
for f in /root/*/.opsfleet-k8s-state /root/.opsfleet-k8s-state; do
  [[ -f "$f" ]] && rm -f "$f"
done

if [[ "$PURGE_CACHE" -eq 1 ]]; then
  echo "=== 清空 /var/cache/opsfleet-k8s ==="
  rm -rf /var/cache/opsfleet-k8s/*
fi

if [[ "$DEEP" -eq 1 ]]; then
  echo "=== --deep：删除集群数据与常见二进制（不可逆）==="
  rm -rf /etc/kubernetes /var/lib/etcd
  rm -f /usr/local/bin/etcd /usr/local/bin/etcdctl
  rm -f /usr/local/bin/kube-apiserver /usr/local/bin/kube-controller-manager \
        /usr/local/bin/kube-scheduler /usr/local/bin/kubectl /usr/local/bin/kubeadm 2>/dev/null || true
fi

echo "=== k8s-offline-test-cleanup 完成 ==="
