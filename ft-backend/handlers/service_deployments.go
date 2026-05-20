package handlers

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"ft-backend/common/response"
	"ft-backend/database"
	"ft-backend/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type serviceDeploymentCreateRequest struct {
	Service       string                 `json:"service" binding:"required"`
	Profile       string                 `json:"profile"`
	InstallMethod string                 `json:"install_method" binding:"required"`
	Version       string                 `json:"version"`
	Params        map[string]interface{} `json:"params"`
	Token         string                 `json:"token"`
}

type serviceDeploymentEventRequest struct {
	Step    string `json:"step" binding:"required"`
	Status  string `json:"status" binding:"required"`
	Message string `json:"message"`
}

type serviceDeploymentFinishRequest struct {
	Status  string `json:"status" binding:"required"`
	Message string `json:"message"`
}

// CreateServiceDeployment stores a deploy spec and returns short executable commands.
func CreateServiceDeployment(c *gin.Context) {
	var req serviceDeploymentCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "无效的服务部署参数")
		return
	}
	normalizeServiceDeploymentRequest(&req)

	token, err := randomToken()
	if err != nil {
		response.ServerError(c, "生成部署 token 失败")
		return
	}
	exp := time.Now().Add(90 * 24 * time.Hour)
	dep := models.ServiceDeployment{
		Service:       req.Service,
		Profile:       req.Profile,
		InstallMethod: req.InstallMethod,
		Version:       req.Version,
		Params:        models.NewJSONBFromMap(req.Params),
		TokenHash:     hashToken(token),
		Status:        "pending",
		CreatedBy:     c.GetString("user_id"),
		ExpiresAt:     &exp,
	}
	if err := database.DB.Create(&dep).Error; err != nil {
		response.ServerError(c, "保存部署规格失败")
		return
	}

	base := publicAPIBaseFromRequest(c)
	id := dep.ID.String()
	curlCmd := fmt.Sprintf("curl -fsSL '%s/api/service-deploy/deployments/%s/bootstrap.sh?token=%s' | sudo bash", base, id, token)
	aiSreCmd := fmt.Sprintf("sudo ai-sre ops service install --api-url %s --deploy-id %s --token %s", quoteShellSingleLine(base), quoteShellSingleLine(id), quoteShellSingleLine(token))
	aiSreUpdateCmd := serviceDeploymentUpdateCommand(req.Service)
	aiSreUninstallCmd := serviceDeploymentUninstallCommand(req.Service)
	aiSreRecoverCmd := serviceDeploymentRecoverCommand(req.Service)
	response.OK(c, gin.H{
		"deploymentId":          id,
		"token":                 token,
		"curlCommand":           curlCmd,
		"aiSreCommand":          aiSreCmd,
		"aiSreUpdateCommand":    aiSreUpdateCmd,
		"aiSreUninstallCommand": aiSreUninstallCmd,
		"aiSreRecoverCommand":   aiSreRecoverCmd,
		"status":                dep.Status,
	})
}

// UpdateServiceDeployment refreshes an existing server-side spec for later ai-sre update.
func UpdateServiceDeployment(c *gin.Context) {
	var req serviceDeploymentCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "无效的服务部署参数")
		return
	}
	normalizeServiceDeploymentRequest(&req)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "无效的部署 ID")
		return
	}
	var dep models.ServiceDeployment
	if err := database.DB.Where("id = ?", id).First(&dep).Error; err != nil {
		response.NotFound(c, "部署任务不存在")
		return
	}
	if dep.Service != req.Service {
		response.BadRequest(c, "部署任务服务类型不匹配")
		return
	}
	exp := time.Now().Add(90 * 24 * time.Hour)
	updates := map[string]interface{}{
		"profile":        req.Profile,
		"install_method": req.InstallMethod,
		"version":        req.Version,
		"params":         models.NewJSONBFromMap(req.Params),
		"status":         "pending_update",
		"current_step":   "spec-update",
		"last_error":     "",
		"finished_at":    nil,
		"expires_at":     &exp,
	}
	if err := database.DB.Model(&dep).Updates(updates).Error; err != nil {
		response.ServerError(c, "更新部署规格失败")
		return
	}
	_ = database.DB.Create(&models.ServiceDeploymentEvent{
		DeploymentID: dep.ID,
		Step:         "spec-update",
		Status:       "success",
		Message:      "deployment spec updated from console",
	}).Error
	base := publicAPIBaseFromRequest(c)
	curlCmd := ""
	aiSreCmd := ""
	if req.Token != "" {
		curlCmd = fmt.Sprintf("curl -fsSL '%s/api/service-deploy/deployments/%s/bootstrap.sh?token=%s' | sudo bash", base, dep.ID.String(), req.Token)
		aiSreCmd = fmt.Sprintf("sudo ai-sre ops service install --api-url %s --deploy-id %s --token %s", quoteShellSingleLine(base), quoteShellSingleLine(dep.ID.String()), quoteShellSingleLine(req.Token))
	}
	response.OK(c, gin.H{
		"deploymentId":       dep.ID.String(),
		"token":              req.Token,
		"curlCommand":        curlCmd,
		"aiSreCommand":       aiSreCmd,
		"aiSreUpdateCommand":    serviceDeploymentUpdateCommand(req.Service),
		"aiSreUninstallCommand": serviceDeploymentUninstallCommand(req.Service),
		"aiSreRecoverCommand":   serviceDeploymentRecoverCommand(req.Service),
		"status":                "pending_update",
	})
}

// ServeServiceDeploymentBootstrap returns a tiny curl|bash entrypoint.
func ServeServiceDeploymentBootstrap(c *gin.Context) {
	dep, token, ok := loadServiceDeploymentByToken(c)
	if !ok {
		return
	}
	base := publicAPIBaseFromRequest(c)
	body := fmt.Sprintf(`#!/usr/bin/env bash
set -euo pipefail
API_BASE=%s
DEPLOY_ID=%s
TOKEN=%s

if ! command -v ai-sre >/dev/null 2>&1; then
  echo "[ai-sre] installing ai-sre client ..."
  curl -fsSL "$API_BASE/api/k8s/deploy/install-ai-sre.sh" | sudo bash
else
  echo "[ai-sre] ai-sre found: $(command -v ai-sre)"
fi

exec ai-sre ops service install --api-url "$API_BASE" --deploy-id "$DEPLOY_ID" --token "$TOKEN"
`, quoteShellSingleLine(base), quoteShellSingleLine(dep.ID.String()), quoteShellSingleLine(token))
	c.Header("Content-Type", "text/plain; charset=utf-8")
	c.Header("Cache-Control", "no-store")
	c.String(http.StatusOK, body)
}

// GetServiceDeploymentSpec returns the complete install spec for ai-sre.
func GetServiceDeploymentSpec(c *gin.Context) {
	dep, _, ok := loadServiceDeploymentByToken(c)
	if !ok {
		return
	}
	var params map[string]interface{}
	if len(dep.Params) > 0 {
		_ = json.Unmarshal([]byte(dep.Params), &params)
	}
	response.OK(c, gin.H{
		"id":             dep.ID.String(),
		"service":        dep.Service,
		"profile":        dep.Profile,
		"install_method": dep.InstallMethod,
		"version":        dep.Version,
		"params":         params,
		"status":         dep.Status,
	})
}

// PostServiceDeploymentEvent appends a progress event.
func PostServiceDeploymentEvent(c *gin.Context) {
	dep, _, ok := loadServiceDeploymentByToken(c)
	if !ok {
		return
	}
	var req serviceDeploymentEventRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "无效的事件参数")
		return
	}
	now := time.Now()
	updates := map[string]interface{}{
		"status":       req.Status,
		"current_step": req.Step,
	}
	if dep.StartedAt == nil && req.Status == "running" {
		updates["started_at"] = now
	}
	if req.Status == "failed" {
		updates["last_error"] = req.Message
	}
	_ = database.DB.Model(&dep).Updates(updates).Error
	ev := models.ServiceDeploymentEvent{
		DeploymentID: dep.ID,
		Step:         req.Step,
		Status:       req.Status,
		Message:      req.Message,
	}
	if err := database.DB.Create(&ev).Error; err != nil {
		response.ServerError(c, "保存部署事件失败")
		return
	}
	response.OK(c, gin.H{"ok": true})
}

// FinishServiceDeployment stores the final result.
func FinishServiceDeployment(c *gin.Context) {
	dep, _, ok := loadServiceDeploymentByToken(c)
	if !ok {
		return
	}
	var req serviceDeploymentFinishRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "无效的完成参数")
		return
	}
	now := time.Now()
	updates := map[string]interface{}{
		"status":      req.Status,
		"finished_at": now,
	}
	if req.Status == "failed" {
		updates["last_error"] = req.Message
	}
	if err := database.DB.Model(&dep).Updates(updates).Error; err != nil {
		response.ServerError(c, "更新部署状态失败")
		return
	}
	if req.Status == "uninstalling" || req.Status == "uninstalled" || req.Status == "uninstall_failed" || req.Status == "pending_uninstall" {
		_ = database.DB.Create(&models.ServiceDeploymentEvent{
			DeploymentID: dep.ID,
			Step:         "uninstall",
			Status:       req.Status,
			Message:      req.Message,
		}).Error
	}
	response.OK(c, gin.H{"ok": true})
}

func normalizeServiceDeploymentRequest(req *serviceDeploymentCreateRequest) {
	req.Service = strings.TrimSpace(strings.ToLower(req.Service))
	req.InstallMethod = strings.TrimSpace(strings.ToLower(req.InstallMethod))
	if req.Profile == "" {
		req.Profile = "default"
	}
	if req.Params == nil {
		req.Params = map[string]interface{}{}
	}
	if req.Version != "" {
		req.Params["version"] = req.Version
	}
}

func serviceDeploymentUpdateCommand(service string) string {
	switch service {
	case "nginx":
		return "sudo ai-sre ops nginx update"
	case "elasticsearch":
		return "sudo ai-sre ops elasticsearch update"
	}
	return ""
}

func serviceDeploymentUninstallCommand(service string) string {
	switch service {
	case "nginx":
		return "sudo ai-sre ops uninstall nginx"
	case "elasticsearch":
		return "sudo ai-sre ops uninstall elasticsearch"
	case "redis", "mysql", "postgresql", "kafka", "haproxy":
		return "sudo ai-sre ops service uninstall " + service
	}
	return ""
}

func serviceDeploymentRecoverCommand(service string) string {
	switch service {
	case "redis", "mysql", "postgresql", "kafka", "haproxy":
		return "sudo ai-sre ops service recover " + service
	}
	return ""
}

func loadServiceDeploymentByToken(c *gin.Context) (models.ServiceDeployment, string, bool) {
	var dep models.ServiceDeployment
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "无效的部署 ID")
		return dep, "", false
	}
	token := strings.TrimSpace(c.Query("token"))
	if token == "" {
		token = strings.TrimSpace(c.GetHeader("X-Deploy-Token"))
	}
	if token == "" {
		response.BadRequest(c, "缺少部署 token")
		return dep, "", false
	}
	if err := database.DB.Where("id = ?", id).First(&dep).Error; err != nil {
		response.NotFound(c, "部署任务不存在")
		return dep, "", false
	}
	if dep.ExpiresAt != nil && time.Now().After(*dep.ExpiresAt) {
		response.BadRequest(c, "部署 token 已过期")
		return dep, "", false
	}
	if dep.TokenHash != hashToken(token) {
		response.BadRequest(c, "部署 token 无效")
		return dep, "", false
	}
	return dep, token, true
}

func randomToken() (string, error) {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// IssueServiceDeploymentPurgeToken creates a short-lived approval token for --purge-data.
func IssueServiceDeploymentPurgeToken(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "无效的部署 ID")
		return
	}
	var dep models.ServiceDeployment
	if err := database.DB.Where("id = ?", id).First(&dep).Error; err != nil {
		response.NotFound(c, "部署任务不存在")
		return
	}
	token, err := randomToken()
	if err != nil {
		response.ServerError(c, "生成 purge token 失败")
		return
	}
	expires := time.Now().Add(15 * time.Minute)
	params := serviceDeploymentParamsMap(dep.Params)
	params["purge_token_hash"] = hashToken(token)
	params["purge_token_expires"] = expires.UTC().Format(time.RFC3339)
	if err := database.DB.Model(&dep).Update("params", models.NewJSONBFromMap(params)).Error; err != nil {
		response.ServerError(c, "保存 purge token 失败")
		return
	}
	response.OK(c, gin.H{
		"purge_token": token,
		"expires_at":  expires.UTC().Format(time.RFC3339),
		"usage":       fmt.Sprintf("sudo ai-sre ops service uninstall %s --purge-data --purge-token %s --yes", dep.Service, token),
	})
}

// VerifyServiceDeploymentPurgeToken validates CLI --purge-token using deployment token auth.
func VerifyServiceDeploymentPurgeToken(c *gin.Context) {
	dep, _, ok := loadServiceDeploymentByToken(c)
	if !ok {
		return
	}
	purgeToken := strings.TrimSpace(c.Query("purge_token"))
	if purgeToken == "" {
		response.BadRequest(c, "缺少 purge_token")
		return
	}
	params := serviceDeploymentParamsMap(dep.Params)
	if hashToken(purgeToken) != strings.TrimSpace(strParam(params, "purge_token_hash")) {
		response.BadRequest(c, "purge token 无效")
		return
	}
	expRaw := strings.TrimSpace(strParam(params, "purge_token_expires"))
	if expRaw == "" {
		response.BadRequest(c, "purge token 未签发")
		return
	}
	exp, err := time.Parse(time.RFC3339, expRaw)
	if err != nil || time.Now().After(exp) {
		response.BadRequest(c, "purge token 已过期")
		return
	}
	response.OK(c, gin.H{"ok": true})
}

func serviceDeploymentParamsMap(raw models.JSONB) map[string]interface{} {
	out := map[string]interface{}{}
	if len(raw) > 0 {
		_ = json.Unmarshal([]byte(raw), &out)
	}
	if out == nil {
		out = map[string]interface{}{}
	}
	return out
}

func strParam(m map[string]interface{}, key string) string {
	if m == nil {
		return ""
	}
	v, _ := m[key].(string)
	return strings.TrimSpace(v)
}
