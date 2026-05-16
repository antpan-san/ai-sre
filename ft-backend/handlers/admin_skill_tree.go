package handlers

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"ft-backend/common/logger"
	"ft-backend/common/response"
	"ft-backend/database"
	"ft-backend/models"
	"ft-backend/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// AdminListSkillTreeVersions lists skill tree revisions.
func AdminListSkillTreeVersions(c *gin.Context) {
	rows, err := services.ListSkillTreeVersions()
	if err != nil {
		logger.Error("AdminListSkillTreeVersions: %v", err)
		response.ServerError(c, "查询技能树版本失败")
		return
	}
	response.OK(c, gin.H{"versions": rows})
}

type adminCreateDraftTreeRequest struct {
	Title string `json:"title"`
	Notes string `json:"notes"`
}

// AdminCreateDraftSkillTree clones the active tree into a new draft revision.
func AdminCreateDraftSkillTree(c *gin.Context) {
	var req adminCreateDraftTreeRequest
	_ = c.ShouldBindJSON(&req)
	active := services.ActiveSkillTree()
	treeRev := fmt.Sprintf("draft.%s", time.Now().UTC().Format("20060102150405"))
	username, _ := c.Get("username")
	name, _ := username.(string)
	var ver models.SkillTreeVersion
	err := database.DB.Transaction(func(tx *gorm.DB) error {
		ver = models.SkillTreeVersion{
			TreeRev: treeRev,
			Status:  models.SkillTreeVersionStatusDraft,
			Title:   strings.TrimSpace(req.Title),
			Notes:   strings.TrimSpace(req.Notes),
		}
		if ver.Title == "" {
			ver.Title = "草稿 " + treeRev
		}
		if err := tx.Create(&ver).Error; err != nil {
			return err
		}
		for _, n := range active.Nodes {
			rec := services.SkillTreeNodeRecordFromService(treeRev, n)
			if err := tx.Create(&rec).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		logger.Error("AdminCreateDraftSkillTree: %v", err)
		response.ServerError(c, "创建草稿版本失败")
		return
	}
	response.OK(c, gin.H{"tree_rev": treeRev, "version": ver, "published_by": name})
}

type adminPublishTreeRequest struct {
	Notes string `json:"notes"`
}

// AdminPublishSkillTreeVersion activates a draft revision (archives previous active).
func AdminPublishSkillTreeVersion(c *gin.Context) {
	treeRev := strings.TrimSpace(c.Param("rev"))
	if treeRev == "" {
		response.BadRequest(c, "tree_rev 无效")
		return
	}
	var req adminPublishTreeRequest
	_ = c.ShouldBindJSON(&req)
	username, _ := c.Get("username")
	name, _ := username.(string)
	now := time.Now().UTC()
	err := database.DB.Transaction(func(tx *gorm.DB) error {
		var draft models.SkillTreeVersion
		if err := tx.Where("tree_rev = ? AND status = ?", treeRev, models.SkillTreeVersionStatusDraft).First(&draft).Error; err != nil {
			return err
		}
		if err := tx.Model(&models.SkillTreeVersion{}).Where("status = ?", models.SkillTreeVersionStatusActive).
			Update("status", models.SkillTreeVersionStatusArchived).Error; err != nil {
			return err
		}
		updates := map[string]interface{}{
			"status":       models.SkillTreeVersionStatusActive,
			"published_by": name,
			"published_at": &now,
		}
		if req.Notes != "" {
			updates["notes"] = req.Notes
		}
		return tx.Model(&draft).Updates(updates).Error
	})
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.NotFound(c, "草稿版本不存在")
			return
		}
		logger.Error("AdminPublishSkillTreeVersion rev=%s: %v", treeRev, err)
		response.ServerError(c, "发布技能树失败")
		return
	}
	response.OK(c, gin.H{"tree_rev": treeRev, "status": models.SkillTreeVersionStatusActive})
}

type adminUpsertTreeNodeRequest struct {
	TreeRev       string `json:"tree_rev" binding:"required"`
	Path          string `json:"path" binding:"required"`
	ParentPath    string `json:"parent_path"`
	NodeType      string `json:"node_type" binding:"required"`
	Title         string `json:"title" binding:"required"`
	Description   string `json:"description"`
	Topic         string `json:"topic"`
	SkillKey      string `json:"skill_key"`
	ProblemKey    string `json:"problem_key"`
	CapabilityKey string `json:"capability_key"`
	PackKey       string `json:"pack_key"`
	FeatureKey    string `json:"feature_key"`
	ExecutionMode string `json:"execution_mode"`
	CLIVisible    *bool  `json:"cli_visible"`
	SortOrder     int    `json:"sort_order"`
}

// AdminCreateSkillTreeNode adds a node (no physical delete policy).
func AdminCreateSkillTreeNode(c *gin.Context) {
	var req adminUpsertTreeNodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "无效参数: "+err.Error())
		return
	}
	if err := services.ValidateSkillTreeNodeInput(req.TreeRev, req.Path, req.ParentPath, req.NodeType, req.SkillKey, ""); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	cliVisible := true
	if req.CLIVisible != nil {
		cliVisible = *req.CLIVisible
	}
	rec := models.SkillTreeNodeRecord{
		TreeRev:       req.TreeRev,
		Path:          strings.TrimSpace(req.Path),
		ParentPath:    strings.TrimSpace(req.ParentPath),
		NodeType:      strings.TrimSpace(req.NodeType),
		Title:         strings.TrimSpace(req.Title),
		Description:   strings.TrimSpace(req.Description),
		Topic:         strings.TrimSpace(req.Topic),
		SkillKey:      strings.TrimSpace(req.SkillKey),
		ProblemKey:    strings.TrimSpace(req.ProblemKey),
		CapabilityKey: strings.TrimSpace(req.CapabilityKey),
		PackKey:       strings.TrimSpace(req.PackKey),
		FeatureKey:    strings.TrimSpace(req.FeatureKey),
		ExecutionMode: strings.TrimSpace(req.ExecutionMode),
		CLIVisible:    cliVisible,
		Status:        models.SkillTreeNodeStatusActive,
		SortOrder:     req.SortOrder,
		Metadata:      models.NewJSONBFromMap(map[string]interface{}{}),
	}
	if err := database.DB.Create(&rec).Error; err != nil {
		logger.Error("AdminCreateSkillTreeNode: %v", err)
		response.ServerError(c, "创建节点失败")
		return
	}
	response.OK(c, gin.H{"node": rec})
}

// AdminUpdateSkillTreeNode edits a node in place.
func AdminUpdateSkillTreeNode(c *gin.Context) {
	id, err := uuid.Parse(strings.TrimSpace(c.Param("id")))
	if err != nil {
		response.BadRequest(c, "无效节点 ID")
		return
	}
	var req adminUpsertTreeNodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "无效参数: "+err.Error())
		return
	}
	var rec models.SkillTreeNodeRecord
	if err := database.DB.Where("id = ?", id).First(&rec).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.NotFound(c, "节点不存在")
			return
		}
		response.ServerError(c, "查询节点失败")
		return
	}
	treeRev := rec.TreeRev
	if err := services.ValidateSkillTreeNodeInput(treeRev, req.Path, req.ParentPath, req.NodeType, req.SkillKey, id.String()); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	updates := map[string]interface{}{
		"path":           strings.TrimSpace(req.Path),
		"parent_path":    strings.TrimSpace(req.ParentPath),
		"node_type":      strings.TrimSpace(req.NodeType),
		"title":          strings.TrimSpace(req.Title),
		"description":    strings.TrimSpace(req.Description),
		"topic":          strings.TrimSpace(req.Topic),
		"skill_key":      strings.TrimSpace(req.SkillKey),
		"problem_key":    strings.TrimSpace(req.ProblemKey),
		"capability_key": strings.TrimSpace(req.CapabilityKey),
		"pack_key":       strings.TrimSpace(req.PackKey),
		"feature_key":    strings.TrimSpace(req.FeatureKey),
		"execution_mode": strings.TrimSpace(req.ExecutionMode),
		"sort_order":     req.SortOrder,
	}
	if req.CLIVisible != nil {
		updates["cli_visible"] = *req.CLIVisible
	}
	if err := database.DB.Model(&rec).Updates(updates).Error; err != nil {
		response.ServerError(c, "更新节点失败")
		return
	}
	response.OK(c, gin.H{"node_id": id.String()})
}

type adminDisableTreeNodeRequest struct {
	Status string `json:"status"` // active | disabled
}

// AdminSetSkillTreeNodeStatus disables or re-enables a node (no delete).
func AdminSetSkillTreeNodeStatus(c *gin.Context) {
	id, err := uuid.Parse(strings.TrimSpace(c.Param("id")))
	if err != nil {
		response.BadRequest(c, "无效节点 ID")
		return
	}
	var req adminDisableTreeNodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "无效参数")
		return
	}
	status := strings.TrimSpace(strings.ToLower(req.Status))
	if status != models.SkillTreeNodeStatusActive && status != models.SkillTreeNodeStatusDisabled {
		response.BadRequest(c, "status 必须为 active 或 disabled")
		return
	}
	if err := database.DB.Model(&models.SkillTreeNodeRecord{}).Where("id = ?", id).Update("status", status).Error; err != nil {
		response.ServerError(c, "更新节点状态失败")
		return
	}
	response.OK(c, gin.H{"node_id": id.String(), "status": status})
}

type adminReorderTreeNode struct {
	ID        string `json:"id" binding:"required"`
	SortOrder int    `json:"sort_order"`
}

type adminReorderTreeRequest struct {
	Items []adminReorderTreeNode `json:"items" binding:"required"`
}

// AdminReorderSkillTreeNodes updates sort_order for multiple nodes.
func AdminReorderSkillTreeNodes(c *gin.Context) {
	var req adminReorderTreeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "无效参数")
		return
	}
	err := database.DB.Transaction(func(tx *gorm.DB) error {
		for _, it := range req.Items {
			id, err := uuid.Parse(strings.TrimSpace(it.ID))
			if err != nil {
				return err
			}
			if err := tx.Model(&models.SkillTreeNodeRecord{}).Where("id = ?", id).Update("sort_order", it.SortOrder).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		response.ServerError(c, "排序更新失败")
		return
	}
	response.OK(c, gin.H{"updated": len(req.Items)})
}

// AdminGetSkillTreeNodes returns nodes for a tree revision (admin edit).
func AdminGetSkillTreeNodes(c *gin.Context) {
	treeRev := strings.TrimSpace(c.Query("tree_rev"))
	if treeRev == "" {
		tree := services.ActiveSkillTree()
		treeRev = tree.TreeRev
	}
	var rows []models.SkillTreeNodeRecord
	if err := database.DB.Where("tree_rev = ?", treeRev).Order("sort_order ASC, path ASC").Find(&rows).Error; err != nil {
		response.ServerError(c, "查询节点失败")
		return
	}
	response.OK(c, gin.H{"tree_rev": treeRev, "nodes": rows})
}
