package handlers

import (
	"crypto/rand"
	"crypto/subtle"
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"ft-backend/common/logger"
	"ft-backend/common/response"
	"ft-backend/database"
	"ft-backend/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

//go:embed k8s_install_bootstrap.sh
var k8sInstallBootstrapSH string

// installRefPrefixV1 与 ai-sre CLI 保持一致：单字符串承载 API 基址 + 资源 UUID + 下载 token。
const installRefPrefixV1 = "ofpk8s1."

type installRefV1 struct {
	B string `json:"b"`
	I string `json:"i"`
	T string `json:"t"`
}

func encodeInstallRefV1(apiBase, id, token string) (string, error) {
	base := strings.TrimRight(strings.TrimSpace(apiBase), "/")
	if base == "" {
		return "", errors.New("empty api base")
	}
	b, err := json.Marshal(installRefV1{B: base, I: id, T: token})
	if err != nil {
		return "", err
	}
	return installRefPrefixV1 + base64.RawURLEncoding.EncodeToString(b), nil
}

// CreateK8sBundleInvite 将当前表单配置登记为可下载资源，返回一键安装引用（无需在目标机上传 zip）。
func CreateK8sBundleInvite(c *gin.Context) {
	var req K8sDeployRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "无效的请求参数: "+err.Error())
		return
	}
	publicBase := strings.TrimSpace(req.PublicAPIBase)
	req.PublicAPIBase = ""

	masters := normalizeHostList(req.MasterHosts)
	if len(masters) == 0 {
		response.BadRequest(c, "请至少填写一个 control plane 节点 IP（masterHosts）")
		return
	}
	if publicBase == "" {
		response.BadRequest(c, "请提供 publicApiBase（浏览器访问的 API 基址，如 https://host:9080/ft-api）")
		return
	}

	payload, err := json.Marshal(req)
	if err != nil {
		response.ServerError(c, "序列化失败")
		return
	}
	tokenBytes := make([]byte, 16)
	if _, err := rand.Read(tokenBytes); err != nil {
		response.ServerError(c, "生成 token 失败")
		return
	}
	token := fmt.Sprintf("%x", tokenBytes)
	exp := time.Now().Add(7 * 24 * time.Hour)

	inv := models.K8sBundleInvite{
		RequestPayload: models.JSONB(payload),
		DownloadToken:  token,
		ExpiresAt:      exp,
	}
	if err := database.DB.Create(&inv).Error; err != nil {
		logger.Error("CreateK8sBundleInvite: %v", err)
		response.ServerError(c, "保存失败")
		return
	}

	ref, err := encodeInstallRefV1(publicBase, inv.ID.String(), token)
	if err != nil {
		response.ServerError(c, "生成安装引用失败")
		return
	}
	cmd := fmt.Sprintf(`sudo ai-sre k8s install '%s'`, ref)
	bootstrap := fmt.Sprintf(`curl -fsSL '%s/api/k8s/deploy/bootstrap.sh' | sudo bash -s -- '%s'`, publicBase, ref)
	response.OK(c, gin.H{
		"id":               inv.ID.String(),
		"expiresAt":        exp.Format(time.RFC3339),
		"installRef":       ref,
		"installCommand":   cmd,
		"bootstrapCommand": bootstrap,
	})
}

// ServeK8sInstallBootstrap 返回可在目标控制机执行的 bash 引导脚本（无需预装 ai-sre，依赖 python3）。
func ServeK8sInstallBootstrap(c *gin.Context) {
	c.Header("Content-Type", "text/plain; charset=utf-8")
	c.Header("Cache-Control", "public, max-age=120")
	c.String(http.StatusOK, k8sInstallBootstrapSH)
}

// DownloadK8sBundleInviteZip 公开下载（凭资源 ID + token），与 GenerateK8sOfflineBundle 产出相同 zip。
func DownloadK8sBundleInviteZip(c *gin.Context) {
	idStr := strings.TrimSpace(c.Param("id"))
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.BadRequest(c, "无效的资源 ID")
		return
	}
	tok := strings.TrimSpace(c.Query("token"))
	if tok == "" {
		tok = strings.TrimSpace(c.GetHeader("X-Opsfleet-Bundle-Token"))
	}
	if tok == "" {
		response.Unauthorized(c, "缺少 token（query token 或 Header X-Opsfleet-Bundle-Token）")
		return
	}

	var inv models.K8sBundleInvite
	if err := database.DB.First(&inv, "id = ?", id).Error; err != nil {
		response.HandleDBError(c, err, "资源不存在或已失效")
		return
	}
	if time.Now().After(inv.ExpiresAt) {
		response.NotFound(c, "安装引用已过期，请在控制台重新生成")
		return
	}
	if subtle.ConstantTimeCompare([]byte(inv.DownloadToken), []byte(tok)) != 1 {
		response.Unauthorized(c, "token 无效")
		return
	}

	var req K8sDeployRequest
	if err := json.Unmarshal(inv.RequestPayload, &req); err != nil {
		logger.Error("bundle invite payload: %v", err)
		response.ServerError(c, "配置数据损坏")
		return
	}

	data, err := BuildK8sOfflineZip(req)
	if err != nil {
		switch {
		case errors.Is(err, ErrK8sBundleMissingMasters):
			response.BadRequest(c, "配置缺少 masterHosts")
			return
		case errors.Is(err, ErrK8sBundleAnsibleDir):
			logger.Error("ansible-agent directory not found for bundle invite")
			response.ServerError(c, "服务器未找到 ansible-agent 目录，无法生成离线包")
			return
		default:
			logger.Error("bundle invite zip: %v", err)
			response.ServerError(c, "打包失败: "+err.Error())
			return
		}
	}

	filename := fmt.Sprintf("opsfleet-k8s-invite-%s.zip", id.String()[:8])
	c.Header("Content-Type", "application/zip")
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	c.Header("Content-Length", fmt.Sprintf("%d", len(data)))
	c.Data(http.StatusOK, "application/zip", data)
}
