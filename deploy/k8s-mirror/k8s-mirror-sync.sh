#!/usr/bin/env bash
# 将 Kubernetes / etcd / CNI 制品拉取到 MIRROR_ROOT，目录布局与 Ansible download URL 一致。
# 支持**多** K8s 版本（与部署页/数据库种子/k8s-mirror-versions.txt 一致）。
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]:-$0}")" && pwd)"
ENV_FILE="${ENV_FILE:-/etc/opsfleet/k8s-mirror.env}"
if [[ -f "$ENV_FILE" ]]; then
  # shellcheck source=/dev/null
  source "$ENV_FILE"
fi

MIRROR_ROOT="${MIRROR_ROOT:-/var/lib/opsfleet-k8s-mirror}"
# 版本文件：可 export KUBERNETES_VERSIONS_FILE；默认脚本同目录，其次 /etc/opsfleet/
_VERSIONS_CAND="${KUBERNETES_VERSIONS_FILE:-}"
if [[ -z "$_VERSIONS_CAND" ]]; then
  for cand in "${SCRIPT_DIR}/k8s-mirror-versions.txt" "/etc/opsfleet/k8s-mirror-versions.txt"; do
    if [[ -f "$cand" ]]; then
      _VERSIONS_CAND=$cand
      break
    fi
  done
fi
VERSIONS_FILE="${_VERSIONS_CAND:-${SCRIPT_DIR}/k8s-mirror-versions.txt}"
KUBERNETES_VERSION="${KUBERNETES_VERSION:-v1.35.4}"
KUBERNETES_ARCHS="${KUBERNETES_ARCHS:-amd64 arm64}"
ETCD_VERSION="${ETCD_VERSION:-v3.6.7}"
ETCD_ARCHS="${ETCD_ARCHS:-amd64 arm64}"
CNI_PLUGINS_VERSION="${CNI_PLUGINS_VERSION:-v1.9.0}"
CNI_ARCHS="${CNI_ARCHS:-amd64 arm64}"
K8S_UPSTREAM="${K8S_UPSTREAM:-https://dl.k8s.io}"
ETCD_UPSTREAM="${ETCD_UPSTREAM:-https://github.com/etcd-io/etcd/releases/download}"
CNI_UPSTREAM="${CNI_UPSTREAM:-https://github.com/containernetworking/plugins/releases/download}"

# 解析要同步的 K8s 版本列表
k8s_versions=()
if [[ -n "${KUBERNETES_VERSIONS:-}" ]]; then
  # shellcheck disable=SC2206
  k8s_versions=(${KUBERNETES_VERSIONS})
elif [[ -f "$VERSIONS_FILE" ]]; then
  while IFS= read -r line || [[ -n "$line" ]]; do
    line="${line#"${line%%[![:space:]]*}"}"
    line="${line%"${line##*[![:space:]]}"}"
    [[ -z "$line" || "$line" == \#* ]] && continue
    k8s_versions+=("$line")
  done <"$VERSIONS_FILE"
else
  k8s_versions=("$KUBERNETES_VERSION")
fi

if [[ ${#k8s_versions[@]} -eq 0 ]]; then
  echo "错误: 未得到任何 K8s 版本（请设置 KUBERNETES_VERSIONS 或提供 $VERSIONS_FILE）"
  exit 1
fi

echo "=== K8s 将同步版本: ${k8s_versions[*]} ==="

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

# --- Kubernetes server tarballs（每个 listed 版本 × arch）---
# 布局：${MIRROR_ROOT}/kubernetes/<ver>/<arch>/kubernetes-server-linux-<arch>.tar.gz
# 与 ansible 内网 URL: .../kubernetes/{{ kubernetes_version }}/{{ arch_version }}/... 一致
for KUBERNETES_VERSION in "${k8s_versions[@]}"; do
  echo "--- kubernetes ${KUBERNETES_VERSION} ---"
  for arch in $KUBERNETES_ARCHS; do
    pkg="kubernetes-server-linux-${arch}.tar.gz"
    rel="kubernetes/${KUBERNETES_VERSION}/${arch}/${pkg}"
    dest="${MIRROR_ROOT}/${rel}"
    url="${K8S_UPSTREAM}/${KUBERNETES_VERSION}/${pkg}"
    dl "$url" "$dest"
    if ! [[ -f "${dest}.sha512" ]]; then
      curl -fsSL --connect-timeout 30 --retry 3 -o "${dest}.sha512" "${url}.sha512" || true
    fi
  done
done

# --- etcd（与 inventory etcd_version 一致，单套版本）---
echo "--- etcd ${ETCD_VERSION} ---"
for arch in $ETCD_ARCHS; do
  pkg="etcd-${ETCD_VERSION}-linux-${arch}.tar.gz"
  rel="etcd/${ETCD_VERSION}/${pkg}"
  dest="${MIRROR_ROOT}/${rel}"
  url="${ETCD_UPSTREAM}/${ETCD_VERSION}/${pkg}"
  dl "$url" "$dest"
done

# --- CNI plugins（单套版本，与 cni_plugins_version 一致）---
echo "--- cni-plugins ${CNI_PLUGINS_VERSION} ---"
for arch in $CNI_ARCHS; do
  pkg="cni-plugins-linux-${arch}-${CNI_PLUGINS_VERSION}.tgz"
  rel="cni-plugins/${CNI_PLUGINS_VERSION}/${pkg}"
  dest="${MIRROR_ROOT}/${rel}"
  url="${CNI_UPSTREAM}/${CNI_PLUGINS_VERSION}/${pkg}"
  dl "$url" "$dest"
done

echo "=== sync 完成: $MIRROR_ROOT (K8s 版本数: ${#k8s_versions[@]}) ==="
