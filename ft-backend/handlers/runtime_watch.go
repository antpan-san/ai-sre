package handlers

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"ft-backend/common/logger"
	"ft-backend/database"
	"ft-backend/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

const runtimeWatchSampleMaxBytes = 512 * 1024

type runtimeWatchSampleBody struct {
	SessionID string          `json:"session_id"`
	Token     string          `json:"token"`
	Watch     json.RawMessage `json:"watch"`
}

func newRuntimeWatchTokenPair() (plain string, hash string, err error) {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return "", "", err
	}
	plain = hex.EncodeToString(b)
	return plain, hashExecutionToken(plain), nil
}

// PostRuntimeWatchSample ingests one watch JSON blob (public, token-authenticated).
func PostRuntimeWatchSample(c *gin.Context) {
	var body runtimeWatchSampleBody
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的请求参数: " + err.Error()})
		return
	}
	sid, err := uuid.Parse(strings.TrimSpace(body.SessionID))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的 session_id"})
		return
	}
	if len(body.Watch) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "watch 不能为空"})
		return
	}
	if len(body.Watch) > runtimeWatchSampleMaxBytes {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "watch 体积过大"})
		return
	}

	var sess models.RuntimeWatchSession
	if err := database.DB.Where("id = ?", sid).First(&sess).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"code": 404, "msg": "会话不存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "查询会话失败"})
		return
	}
	if sess.Status != "active" {
		c.JSON(http.StatusConflict, gin.H{"code": 409, "msg": "会话已结束"})
		return
	}
	if sess.SampleTokenHash == "" || subtle.ConstantTimeCompare([]byte(sess.SampleTokenHash), []byte(hashExecutionToken(body.Token))) != 1 {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "msg": "令牌无效"})
		return
	}

	row := models.RuntimeWatchSample{
		SessionID:  sid,
		ObservedAt: time.Now().UTC(),
		Payload:    models.JSONB(append(json.RawMessage(nil), body.Watch...)),
	}
	if err := database.DB.Create(&row).Error; err != nil {
		logger.Error("runtime watch sample insert: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "保存样本失败"})
		return
	}
	if watch := extractDiagnosisFromWatchJSON(body.Watch); watch != nil {
		applyWatchDiagnosisToSession(&sess, watch)
		_ = database.DB.Model(&sess).Updates(map[string]interface{}{
			"target_display":    sess.TargetDisplay,
			"resource_kind":     sess.ResourceKind,
			"resource_name":     sess.ResourceName,
			"work_pod":          sess.WorkPod,
			"diagnosis_level":   sess.DiagnosisLevel,
			"root_cause":        sess.RootCause,
			"evidence":          sess.Evidence,
			"diagnosis_source":  sess.DiagnosisSource,
			"sample_count":      sess.SampleCount,
			"last_diagnosed_at": sess.LastDiagnosedAt,
		}).Error
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "msg": "样本已接收", "data": gin.H{"id": row.ID}})
}

type createRuntimeWatchSessionBody struct {
	Namespace   string `json:"namespace" binding:"required"`
	Pod         string `json:"pod" binding:"required"`
	Container   string `json:"container"`
	IntervalSec int    `json:"interval_sec"`
	MachineNote string `json:"machine_note"`
}

// CreateRuntimeWatchSession creates a session and returns a one-time sample write token.
func CreateRuntimeWatchSession(c *gin.Context) {
	uid := models.UserIDFromContext(c.MustGet("userID"))
	if uid == uuid.Nil {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "msg": "未授权"})
		return
	}
	var body createRuntimeWatchSessionBody
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的请求参数: " + err.Error()})
		return
	}
	if body.IntervalSec <= 0 {
		body.IntervalSec = 15
	}
	if body.IntervalSec < 5 {
		body.IntervalSec = 5
	}
	if body.IntervalSec > 3600 {
		body.IntervalSec = 3600
	}
	plain, hash, err := newRuntimeWatchTokenPair()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "生成令牌失败"})
		return
	}
	sess := models.RuntimeWatchSession{
		UserID:          uid,
		Namespace:       strings.TrimSpace(body.Namespace),
		Pod:             strings.TrimSpace(body.Pod),
		Container:       strings.TrimSpace(body.Container),
		IntervalSec:     body.IntervalSec,
		Status:          "active",
		SampleTokenHash: hash,
		MachineNote:     strings.TrimSpace(body.MachineNote),
	}
	if err := database.DB.Create(&sess).Error; err != nil {
		logger.Error("runtime watch session create: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "创建会话失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "success",
		"data": gin.H{
			"id":                 sess.ID,
			"sample_write_token": plain,
			"upload_path":        "/api/runtime-watch/sample",
			"namespace":          sess.Namespace,
			"pod":                sess.Pod,
			"container":          sess.Container,
			"interval_sec":       sess.IntervalSec,
		},
	})
}

// ListRuntimeWatchSessions lists sessions for the current user.
func ListRuntimeWatchSessions(c *gin.Context) {
	uid := models.UserIDFromContext(c.MustGet("userID"))
	if uid == uuid.Nil {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "msg": "未授权"})
		return
	}
	var rows []models.RuntimeWatchSession
	// 仅列出 CLI 单次 diagnose 上传的报告（不含手动创建的持续采样空会话）
	if err := database.DB.Where("user_id = ? AND root_cause <> ''", uid).
		Order("COALESCE(last_diagnosed_at, created_at) DESC").
		Limit(200).Find(&rows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "查询失败"})
		return
	}
	out := make([]gin.H, 0, len(rows))
	for _, r := range rows {
		row := sessionDiagnosisRow(r)
		if !row["has_diagnosis"].(bool) {
			var sample models.RuntimeWatchSample
			if err := database.DB.Where("session_id = ?", r.ID).Order("observed_at DESC").First(&sample).Error; err == nil {
				if watch := extractDiagnosisFromWatchJSON([]byte(sample.Payload)); watch != nil {
					applyWatchDiagnosisToSession(&r, watch)
					_ = database.DB.Model(&r).Updates(map[string]interface{}{
						"target_display": r.TargetDisplay, "resource_kind": r.ResourceKind,
						"resource_name": r.ResourceName, "work_pod": r.WorkPod,
						"diagnosis_level": r.DiagnosisLevel, "root_cause": r.RootCause,
						"evidence": r.Evidence, "diagnosis_source": r.DiagnosisSource,
						"sample_count": r.SampleCount, "last_diagnosed_at": r.LastDiagnosedAt,
					}).Error
					row = sessionDiagnosisRow(r)
				}
			}
		}
		out = append(out, row)
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "msg": "success", "data": out})
}

// GetRuntimeWatchSamples returns recent samples for a session.
func GetRuntimeWatchSamples(c *gin.Context) {
	uid := models.UserIDFromContext(c.MustGet("userID"))
	if uid == uuid.Nil {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "msg": "未授权"})
		return
	}
	sid, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的 id"})
		return
	}
	var sess models.RuntimeWatchSession
	if err := database.DB.Where("id = ? AND user_id = ?", sid, uid).First(&sess).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"code": 404, "msg": "会话不存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "查询失败"})
		return
	}
	q := database.DB.Where("session_id = ?", sid).Order("observed_at ASC")
	if since := strings.TrimSpace(c.Query("since")); since != "" {
		if t, err := time.Parse(time.RFC3339, since); err == nil {
			q = q.Where("observed_at > ?", t)
		}
	}
	var samples []models.RuntimeWatchSample
	if err := q.Limit(500).Find(&samples).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "查询样本失败"})
		return
	}
	out := make([]gin.H, 0, len(samples))
	for _, s := range samples {
		out = append(out, gin.H{
			"id":          s.ID,
			"observed_at": s.ObservedAt,
			"payload":     json.RawMessage(s.Payload),
		})
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "msg": "success", "data": gin.H{
		"session": sessionDiagnosisRow(sess),
		"samples": out,
	}})
}

// DeleteRuntimeWatchSession deletes a session owned by the current user and its samples (hard delete).
func DeleteRuntimeWatchSession(c *gin.Context) {
	uid := models.UserIDFromContext(c.MustGet("userID"))
	if uid == uuid.Nil {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "msg": "未授权"})
		return
	}
	sid, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的 id"})
		return
	}
	err = database.DB.Transaction(func(tx *gorm.DB) error {
		var sess models.RuntimeWatchSession
		if err := tx.Where("id = ? AND user_id = ?", sid, uid).First(&sess).Error; err != nil {
			return err
		}
		if err := tx.Where("session_id = ?", sid).Delete(&models.RuntimeWatchSample{}).Error; err != nil {
			return err
		}
		return tx.Where("id = ? AND user_id = ?", sid, uid).Delete(&models.RuntimeWatchSession{}).Error
	})
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"code": 404, "msg": "会话不存在"})
			return
		}
		logger.Error("runtime watch session delete: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "删除失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "msg": "已删除"})
}

// StopRuntimeWatchSession marks a session stopped.
func StopRuntimeWatchSession(c *gin.Context) {
	uid := models.UserIDFromContext(c.MustGet("userID"))
	if uid == uuid.Nil {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "msg": "未授权"})
		return
	}
	sid, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的 id"})
		return
	}
	tx := database.DB.Model(&models.RuntimeWatchSession{}).
		Where("id = ? AND user_id = ?", sid, uid).
		Update("status", "stopped")
	if tx.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "更新失败"})
		return
	}
	if tx.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "msg": "会话不存在"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "msg": "已停止"})
}
