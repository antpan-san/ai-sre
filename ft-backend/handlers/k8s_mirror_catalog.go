package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"ft-backend/common/logger"
	"ft-backend/common/response"

	"github.com/gin-gonic/gin"
)

// K8sMirrorManifest 与 deploy/k8s-mirror/k8s-mirror-generate-manifest.sh 产出的 manifest.json 对齐。
type K8sMirrorManifest struct {
	GeneratedAt   string            `json:"generatedAt"`
	MirrorRoot    string            `json:"mirrorRoot"`
	PublicBaseURL string            `json:"publicBaseUrl"`
	Files         []K8sMirrorFile   `json:"files"`
	FetchError    string            `json:"fetchError,omitempty"`
	ManifestURL   string            `json:"manifestUrl,omitempty"`
}

type K8sMirrorFile struct {
	RelativePath string `json:"relativePath"`
	SizeBytes    int64  `json:"sizeBytes"`
	SHA512       string `json:"sha512"`
	DownloadURL  string `json:"downloadUrl"`
}

func resolveK8sMirrorManifestURL() string {
	if u := strings.TrimSpace(os.Getenv("OPSFLEET_K8S_MIRROR_MANIFEST_URL")); u != "" {
		return u
	}
	base := strings.TrimSpace(os.Getenv("OPSFLEET_K8S_MIRROR_BASE_URL"))
	if base == "" {
		base = "http://192.168.56.11"
	}
	return strings.TrimRight(base, "/") + "/manifest.json"
}

// GetK8sMirrorCatalog 代理拉取制品站 manifest.json，供前端展示 SHA 与下载路径。
func GetK8sMirrorCatalog(c *gin.Context) {
	url := resolveK8sMirrorManifestURL()
	client := &http.Client{Timeout: 45 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		logger.Warn("k8s mirror manifest fetch: %v", err)
		response.OK(c, K8sMirrorManifest{
			FetchError:  fmt.Sprintf("无法拉取 manifest: %v", err),
			ManifestURL: url,
		})
		return
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 64<<20))
	if err != nil {
		response.OK(c, K8sMirrorManifest{
			FetchError:  fmt.Sprintf("读取 manifest 失败: %v", err),
			ManifestURL: url,
		})
		return
	}
	if resp.StatusCode != http.StatusOK {
		response.OK(c, K8sMirrorManifest{
			FetchError:  fmt.Sprintf("HTTP %d: %s", resp.StatusCode, truncateRunes(string(body), 500)),
			ManifestURL: url,
		})
		return
	}
	var m K8sMirrorManifest
	if err := json.Unmarshal(body, &m); err != nil {
		response.OK(c, K8sMirrorManifest{
			FetchError:  fmt.Sprintf("manifest JSON 解析失败: %v", err),
			ManifestURL: url,
		})
		return
	}
	m.ManifestURL = url
	response.OK(c, m)
}

func truncateRunes(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max]) + "…"
}
