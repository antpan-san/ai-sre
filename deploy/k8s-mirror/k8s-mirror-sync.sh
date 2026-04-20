#!/usr/bin/env bash
# 将 Kubernetes / etcd / CNI 制品拉取到 MIRROR_ROOT，目录布局与 Ansible download URL 一致。
set -euo pipefail

ENV_FILE="${ENV_FILE:-/etc/opsfleet/k8s-mirror.env}"
if [[ -f "$ENV_FILE" ]]; then
  # shellcheck source=/dev/null
  source "$ENV_FILE"
fi

MIRROR_ROOT="${MIRROR_ROOT:-/var/lib/opsfleet-k8s-mirror}"
KUBERNETES_VERSION="${KUBERNETES_VERSION:-v1.35.0}"
KUBERNETES_ARCHS="${KUBERNETES_ARCHS:-amd64 arm64}"
ETCD_VERSION="${ETCD_VERSION:-v3.6.7}"
ETCD_ARCHS="${ETCD_ARCHS:-amd64 arm64}"
CNI_PLUGINS_VERSION="${CNI_PLUGINS_VERSION:-v1.9.0}"
CNI_ARCHS="${CNI_ARCHS:-amd64 arm64}"
K8S_UPSTREAM="${K8S_UPSTREAM:-https://dl.k8s.io}"
ETCD_UPSTREAM="${ETCD_UPSTREAM:-https://github.com/etcd-io/etcd/releases/download}"
CNI_UPSTREAM="${CNI_UPSTREAM:-https://github.com/containernetworking/plugins/releases/download}"

mkdir -p "$MIRROR_ROOT"

dl() {
  local url="$1" dest="$2"
  mkdir -p "$(dirname "$dest")"
  if [[ -f "$dest" ]]; then
    echo "  exists skip: $dest"
    return 0
  fi
  echo "  GET $url -> $dest"
  curl -fsSL --connect-timeout 30 --retry 3 -o "$dest" "$url"
}

# --- Kubernetes server tarball ---
for arch in $KUBERNETES_ARCHS; do
  pkg="kubernetes-server-linux-${arch}.tar.gz"
  rel="kubernetes/${KUBERNETES_VERSION}/${arch}/${pkg}"
  dest="${MIRROR_ROOT}/${rel}"
  url="${K8S_UPSTREAM}/${KUBERNETES_VERSION}/${arch}/${pkg}"
  dl "$url" "$dest"
  # 官方 sha512 旁路保存（可选校验）
  if ! [[ -f "${dest}.sha512" ]]; then
    curl -fsSL --connect-timeout 30 --retry 3 -o "${dest}.sha512" "${url}.sha512" || true
  fi
done

# --- etcd ---
for arch in $ETCD_ARCHS; do
  pkg="etcd-${ETCD_VERSION}-linux-${arch}.tar.gz"
  rel="etcd/${ETCD_VERSION}/${pkg}"
  dest="${MIRROR_ROOT}/${rel}"
  url="${ETCD_UPSTREAM}/${ETCD_VERSION}/${pkg}"
  dl "$url" "$dest"
done

# --- CNI plugins ---
for arch in $CNI_ARCHS; do
  pkg="cni-plugins-linux-${arch}-${CNI_PLUGINS_VERSION}.tgz"
  rel="cni-plugins/${CNI_PLUGINS_VERSION}/${pkg}"
  dest="${MIRROR_ROOT}/${rel}"
  url="${CNI_UPSTREAM}/${CNI_PLUGINS_VERSION}/${pkg}"
  dl "$url" "$dest"
done

echo "=== sync 完成: $MIRROR_ROOT ==="
