package handlers

import (
	"ft-backend/common/response"

	"github.com/gin-gonic/gin"
)

// K8sComponentImageRow 与 ansible-agent/inventory 及 k8s_addons 模板中的镜像版本对齐，供内网预拉与对照。
// 更新镜像时须同步改 ansible 与前端引用（以本处与 group_vars 为单一事实来源的 HTTP 面）。
type K8sComponentImageRow struct {
	Component  string `json:"component"`
	VersionKey string `json:"versionKey"`
	Version    string `json:"version"`
	Image      string `json:"image"`
	Notes      string `json:"notes"`
}

// k8sComponentCatalogRows 与当前仓库 ansible-agent 默认值一致；合并包内 overlay 可覆盖 coredns_image / pause_image 等。
var k8sComponentCatalogRows = []K8sComponentImageRow{
	{Component: "pause (sandbox)", VersionKey: "pause", Version: "3.10", Image: "registry.k8s.io/pause:3.10", Notes: "kubelet pod 沙箱；可在页面「Pause 镜像」覆盖"},
	{Component: "CoreDNS", VersionKey: "coredns_version", Version: "v1.13.1", Image: "registry.k8s.io/coredns/coredns:v1.13.1", Notes: "k8s_addons 模板 coredns.yaml.j2"},
	{Component: "Flannel (CNI init)", VersionKey: "flannel_cni", Version: "v1.4.1-flannel1", Image: "docker.io/flannel/flannel-cni-plugin:v1.4.1-flannel1", Notes: "network_plugin=flannel 时"},
	{Component: "Flannel (main)", VersionKey: "flannel", Version: "v0.25.2", Image: "docker.io/flannel/flannel:v0.25.2", Notes: "network_plugin=flannel 时"},
	{Component: "Calico CNI", VersionKey: "calico", Version: "v3.31.3", Image: "quay.io/calico/cni:v3.31.3", Notes: "network_plugin=calico 时（官方 manifest 内）"},
	{Component: "Calico node", VersionKey: "calico", Version: "v3.31.3", Image: "quay.io/calico/node:v3.31.3", Notes: "network_plugin=calico 时"},
	{Component: "Calico kube-controllers", VersionKey: "calico", Version: "v3.31.3", Image: "quay.io/calico/kube-controllers:v3.31.3", Notes: "network_plugin=calico 时"},
}

// K8sComponentCatalogDoc 附加说明行（无镜像地址）。
var k8sComponentCatalogDoc = []map[string]string{
	{"key": "calico_manifest", "value": "https://raw.githubusercontent.com/projectcalico/calico/v3.31.3/manifests/calico.yaml", "description": "Calico 安装前下载并打补丁；内网可镜像到本仓库 download_domain 后设 group_vars：calico_manifest_url"},
	{"key": "inventory_calico_version", "value": "v3.31.3", "description": "与 ansible inventory group_vars 中 calico_version 一致"},
}

// GetK8sComponentCatalog 返回 CNI / DNS / pause 等默认镜像与版本，便于内网预拉与「页面展示」与 inventory 一致。
func GetK8sComponentCatalog(c *gin.Context) {
	response.OK(c, gin.H{
		"images": k8sComponentCatalogRows,
		"docs":   k8sComponentCatalogDoc,
		"networkPluginsSupported": []string{
			"flannel",
			"calico",
		},
		"notImplemented": []string{"cilium", "weave"},
	})
}
