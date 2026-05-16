package handlers

import (
	"errors"
	"strconv"
	"strings"

	"ft-backend/common/logger"
	"ft-backend/common/response"
	"ft-backend/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// AdminListSkillAssets lists skill assets for super_admin review.
func AdminListSkillAssets(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	items, total, err := services.ListSkillAssets(c.Query("status"), c.Query("topic"), page, pageSize)
	if err != nil {
		logger.Error("AdminListSkillAssets: %v", err)
		response.ServerError(c, "查询技能资产失败")
		return
	}
	response.OK(c, gin.H{
		"items":     items,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// AdminGetSkillAsset returns one skill asset with version content.
func AdminGetSkillAsset(c *gin.Context) {
	id, err := uuid.Parse(strings.TrimSpace(c.Param("id")))
	if err != nil {
		response.BadRequest(c, "无效资产 ID")
		return
	}
	detail, err := services.GetSkillAssetDetail(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.NotFound(c, "技能资产不存在")
			return
		}
		logger.Error("AdminGetSkillAsset: %v", err)
		response.ServerError(c, "查询技能资产失败")
		return
	}
	response.OK(c, gin.H{"asset": detail})
}

type adminSkillAssetApproveRequest struct {
	Notes string `json:"notes"`
}

// AdminApproveSkillAsset publishes an approved skill pack to the registry data dir.
func AdminApproveSkillAsset(c *gin.Context) {
	id, err := uuid.Parse(strings.TrimSpace(c.Param("id")))
	if err != nil {
		response.BadRequest(c, "无效资产 ID")
		return
	}
	var req adminSkillAssetApproveRequest
	_ = c.ShouldBindJSON(&req)
	adminID, _ := c.Get("userID")
	adminName, _ := c.Get("username")
	uid, _ := adminID.(uuid.UUID)
	name, _ := adminName.(string)
	pack, path, err := services.ApproveSkillAsset(id, uid, name, strings.TrimSpace(req.Notes))
	if err != nil {
		switch err.Error() {
		case "already_approved":
			response.BadRequest(c, "该技能资产已审核通过")
		case "no_version":
			response.BadRequest(c, "技能资产缺少版本内容")
		case "invalid_pack":
			response.ServerError(c, "生成的技能包不符合 schema")
		default:
			if errors.Is(err, gorm.ErrRecordNotFound) {
				response.NotFound(c, "技能资产不存在")
				return
			}
			logger.Error("AdminApproveSkillAsset id=%s: %v", id, err)
			if strings.Contains(err.Error(), "OPSFLEET_AI_SKILL_DATA_DIR") {
				response.ServerError(c, err.Error())
				return
			}
			response.ServerError(c, "审核通过失败: "+err.Error())
		}
		return
	}
	if path == "" {
		response.ServerError(c, "技能包写入注册表失败")
		return
	}
	response.OK(c, gin.H{
		"asset_id": id.String(),
		"status":   "approved",
		"pack":     pack,
		"path":     path,
	})
}

type adminSkillAssetRejectRequest struct {
	Reason string `json:"reason"`
}

// AdminRejectSkillAsset deprecates a pending skill asset.
func AdminRejectSkillAsset(c *gin.Context) {
	id, err := uuid.Parse(strings.TrimSpace(c.Param("id")))
	if err != nil {
		response.BadRequest(c, "无效资产 ID")
		return
	}
	var req adminSkillAssetRejectRequest
	_ = c.ShouldBindJSON(&req)
	adminName, _ := c.Get("username")
	name, _ := adminName.(string)
	if err := services.RejectSkillAsset(id, name, strings.TrimSpace(req.Reason)); err != nil {
		switch err.Error() {
		case "already_approved":
			response.BadRequest(c, "已审核通过的技能资产不能驳回")
		default:
			if errors.Is(err, gorm.ErrRecordNotFound) {
				response.NotFound(c, "技能资产不存在")
				return
			}
			logger.Error("AdminRejectSkillAsset id=%s: %v", id, err)
			response.ServerError(c, "驳回失败")
		}
		return
	}
	response.OK(c, gin.H{"asset_id": id.String(), "status": "deprecated"})
}
