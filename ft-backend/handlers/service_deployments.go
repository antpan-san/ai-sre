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

	token, err := randomToken()
	if err != nil {
		response.ServerError(c, "生成部署 token 失败")
		return
	}
	exp := time.Now().Add(24 * time.Hour)
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
	aiSreCmd := fmt.Sprintf("sudo ai-sre service install --api-url %s --deploy-id %s --token %s", quoteShellSingleLine(base), quoteShellSingleLine(id), quoteShellSingleLine(token))
	response.OK(c, gin.H{
		"deploymentId": id,
		"token":        token,
		"curlCommand":  curlCmd,
		"aiSreCommand": aiSreCmd,
		"status":       dep.Status,
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

exec ai-sre service install --api-url "$API_BASE" --deploy-id "$DEPLOY_ID" --token "$TOKEN"
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
	response.OK(c, gin.H{"ok": true})
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
