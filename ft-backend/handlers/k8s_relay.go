package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"ft-backend/common/logger"
	"ft-backend/common/response"

	"github.com/gin-gonic/gin"
)

// resolveK8sRelayBaseURL 中转/制品 HTTP 根（无尾斜杠），用于 warm 与预检。
func resolveK8sRelayBaseURL() string {
	if b := strings.TrimSpace(os.Getenv("OPSFLEET_K8S_RELAY_BASE_URL")); b != "" {
		return strings.TrimRight(b, "/")
	}
	if b := strings.TrimSpace(os.Getenv("OPSFLEET_K8S_MIRROR_BASE_URL")); b != "" {
		return strings.TrimRight(b, "/")
	}
	return strings.TrimRight("http://192.168.56.11", "/")
}

// K8sRelayPreflightResponse 控制台「生成命令」时只读预检（不下载大文件）。
type K8sRelayPreflightResponse struct {
	RelayBaseURL         string `json:"relayBaseUrl"`
	MirrorManifestURL    string `json:"mirrorManifestUrl"`
	RelayHealthOK        bool   `json:"relayHealthOk"`
	RelayHealthError     string `json:"relayHealthError,omitempty"`
	MirrorManifestOK     bool   `json:"mirrorManifestOk"`
	MirrorManifestError  string `json:"mirrorManifestError,omitempty"`
	PrimaryProbeURL      string `json:"primaryProbeUrl,omitempty"`
	PrimaryProbeOK       bool   `json:"primaryProbeOk"`
	PrimaryProbeError    string `json:"primaryProbeError,omitempty"`
	Notes                string `json:"notes"`
}

// GetK8sRelayPreflight GET /api/k8s/deploy/relay/preflight — 轻量探测中转与（可选）公网 tarball Range GET（bytes=0-0）。
func GetK8sRelayPreflight(c *gin.Context) {
	base := resolveK8sRelayBaseURL()
	manURL := resolveK8sMirrorManifestURL()
	out := K8sRelayPreflightResponse{
		RelayBaseURL:      base,
		MirrorManifestURL: manURL,
		Notes:             "生成离线包不会 warm 大文件；执行 install.sh 时可选客户端探测后追加 relay overlay。服务端探测路径与客户机可能不一致。",
	}

	client := &http.Client{Timeout: 8 * time.Second}
	if hr, err := client.Get(base + "/health"); err == nil {
		io.Copy(io.Discard, hr.Body)
		hr.Body.Close()
		out.RelayHealthOK = hr.StatusCode == http.StatusOK
		if !out.RelayHealthOK {
			out.RelayHealthError = fmt.Sprintf("HTTP %d", hr.StatusCode)
		}
	} else {
		out.RelayHealthError = err.Error()
	}

	mClient := &http.Client{Timeout: 12 * time.Second}
	if mr, err := mClient.Get(manURL); err == nil {
		b, _ := io.ReadAll(io.LimitReader(mr.Body, 1<<20))
		mr.Body.Close()
		out.MirrorManifestOK = mr.StatusCode == http.StatusOK && json.Valid(b)
		if mr.StatusCode != http.StatusOK {
			out.MirrorManifestError = fmt.Sprintf("HTTP %d", mr.StatusCode)
		}
	} else {
		out.MirrorManifestError = err.Error()
	}

	if u := strings.TrimSpace(c.Query("primary_probe_url")); u != "" {
		out.PrimaryProbeURL = u
		pu, err := url.Parse(u)
		if err == nil && pu.Scheme != "" && pu.Host != "" {
			pClient := &http.Client{Timeout: 12 * time.Second}
			req, _ := http.NewRequest(http.MethodGet, u, nil)
			req.Header.Set("Range", "bytes=0-0")
			if pr, err := pClient.Do(req); err == nil {
				io.Copy(io.Discard, pr.Body)
				pr.Body.Close()
				out.PrimaryProbeOK = pr.StatusCode == http.StatusOK || pr.StatusCode == http.StatusPartialContent
				if !out.PrimaryProbeOK {
					out.PrimaryProbeError = fmt.Sprintf("HTTP %d", pr.StatusCode)
				}
			} else {
				out.PrimaryProbeError = err.Error()
			}
		}
	}

	response.OK(c, out)
}

// K8sRelayWarmRequest 执行 install 前可选触发中转按需拉取（幂等）。
type K8sRelayWarmRequest struct {
	Paths []string `json:"paths"`
}

// PostK8sRelayWarm POST /api/k8s/deploy/relay/warm — 对 relay 上相对路径发起 GET 触发 mirror-serve 回源（仅在脚本已执行时调用）。
func PostK8sRelayWarm(c *gin.Context) {
	var req K8sRelayWarmRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "无效 JSON: "+err.Error())
		return
	}
	base := resolveK8sRelayBaseURL()
	client := &http.Client{Timeout: 45 * time.Minute}
	results := make([]map[string]any, 0, len(req.Paths))
	for _, rel := range req.Paths {
		rel = strings.TrimSpace(rel)
		if rel == "" || strings.Contains(rel, "..") {
			results = append(results, map[string]any{"path": rel, "ok": false, "error": "invalid path"})
			continue
		}
		if !strings.HasPrefix(rel, "/") {
			rel = "/" + rel
		}
		u := base + rel
		resp, err := client.Get(u)
		if err != nil {
			logger.Warn("relay warm GET %s: %v", u, err)
			results = append(results, map[string]any{"path": rel, "ok": false, "error": err.Error()})
			continue
		}
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
		ok := resp.StatusCode == http.StatusOK
		results = append(results, map[string]any{"path": rel, "ok": ok, "status": resp.StatusCode})
	}
	response.OK(c, map[string]any{"relayBaseUrl": base, "results": results})
}

// buildResourceRoutingForOfflineBundle 为「阿里云」镜像源生成客户端公网探测 + relay 覆盖片段。
func buildResourceRoutingForOfflineBundle(req K8sDeployRequest) (jsonOut string, relayOverlay string) {
	if normalizeImageSource(req.ImageSource) != "aliyun" {
		b, _ := json.Marshal(map[string]any{
			"routing_enabled": false,
			"mode":            "mirror_only",
			"notes":           "非 aliyun 镜像源时 tarball 已走 download_domain，无需公网/relay 双路径客户端路由。",
		})
		return string(b), ""
	}
	ver := strings.TrimSpace(req.Version)
	if ver == "" {
		ver = "v1.35.4"
	}
	arch := normalizeK8sCPUArch(req.ArchVersion)
	pkg := fmt.Sprintf("kubernetes-server-linux-%s.tar.gz", arch)
	primary := fmt.Sprintf("https://dl.k8s.io/%s/%s", ver, pkg)

	relayProto := "http://"
	if p := strings.TrimSpace(req.DownloadProtocol); p != "" {
		relayProto = normalizeDownloadProtocol(p)
	}
	relayHost := strings.TrimSpace(req.DownloadDomain)
	if relayHost == "" {
		if b := strings.TrimSpace(os.Getenv("OPSFLEET_K8S_MIRROR_BASE_URL")); b != "" {
			if u, err := url.Parse(b); err == nil && u.Host != "" {
				relayHost = u.Host
				if u.Scheme != "" {
					relayProto = normalizeDownloadProtocol(u.Scheme + "://")
				}
			}
		}
	}
	if relayHost == "" {
		relayHost = "192.168.56.11"
	}

	relayOverlay = fmt.Sprintf(`# --- OpsFleet client resource routing (append when dl.k8s.io unreachable on control host) ---
image_source: "default"
download_protocol: "%s"
download_domain: "%s"
k8s_server_tarball_url: "{{ download_protocol }}{{ download_domain }}/kubernetes/{{ kubernetes_version }}/{{ arch_version }}/{{ k8s_package_name }}.tar.gz"
k8s_server_tarball_checksum: "sha512:{{ download_protocol }}{{ download_domain }}/kubernetes/{{ kubernetes_version }}/{{ arch_version }}/{{ k8s_package_name }}.tar.gz.sha512"
etcd_download_url: "{{ download_protocol }}{{ download_domain }}/etcd/{{ etcd_version }}/{{ etcd_package_name }}.tar.gz"
cni_plugins_download_url: "{{ download_protocol }}{{ download_domain }}/cni-plugins/{{ cni_plugins_version }}/cni-plugins-linux-{{ arch_version }}-{{ cni_plugins_version }}.tgz"
`, relayProto, relayHost)

	doc := map[string]any{
		"routing_enabled":        true,
		"mode":                   "aliyun_with_relay_fallback",
		"primary_probe_url":      primary,
		"probe_connect_seconds":  5,
		"relay_download_domain": relayHost,
		"relay_download_protocol": relayProto,
		"notes": "执行 install.sh 时若公网 Range GET（bytes=0-0）探测失败，将把 resource_routing_relay_overlay.yml 追加进 inventory/group_vars/all.yml 后走内网制品；大文件仍仅在执行阶段下载。",
	}
	b, _ := json.MarshalIndent(doc, "", "  ")
	return string(b), relayOverlay
}
