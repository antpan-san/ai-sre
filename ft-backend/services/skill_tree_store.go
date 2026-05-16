package services

import (
	"errors"
	"fmt"
	"strings"
	"sync"

	"ft-backend/common/logger"
	"ft-backend/database"
	"ft-backend/models"

	"gorm.io/gorm"
)

const (
	BuiltinSkillTreeRev = "builtin.skill-tree.v1"
)

// ActiveSkillTreeResult is the resolved tree for runtime reads.
type ActiveSkillTreeResult struct {
	TreeRev string
	Source  string // "database" | "builtin"
	Nodes   []SkillTreeNode
}

var (
	seedSkillTreeOnce sync.Once
)

// SeedBuiltinSkillTree ensures the builtin tree revision exists in the database.
// Safe to call on every startup; uses IF NOT EXISTS semantics via idempotent checks.
func SeedBuiltinSkillTree() error {
	var seedErr error
	seedSkillTreeOnce.Do(func() {
		seedErr = seedBuiltinSkillTreeOnce()
	})
	return seedErr
}

func seedBuiltinSkillTreeOnce() error {
	if database.DB == nil {
		return nil
	}
	var active models.SkillTreeVersion
	err := database.DB.Where("status = ?", models.SkillTreeVersionStatusActive).First(&active).Error
	if err == nil {
		return nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	return database.DB.Transaction(func(tx *gorm.DB) error {
		var existing models.SkillTreeVersion
		if err := tx.Where("tree_rev = ?", BuiltinSkillTreeRev).First(&existing).Error; err == nil {
			if existing.Status != models.SkillTreeVersionStatusActive {
				return tx.Model(&existing).Update("status", models.SkillTreeVersionStatusActive).Error
			}
			return ensureBuiltinNodes(tx, BuiltinSkillTreeRev)
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		ver := models.SkillTreeVersion{
			TreeRev:     BuiltinSkillTreeRev,
			Status:      models.SkillTreeVersionStatusActive,
			Title:       "内置技能树 v1",
			Notes:       "从代码内置树 seed；数据库异常时仍可用 builtin fallback",
			PublishedBy: "system",
		}
		if err := tx.Create(&ver).Error; err != nil {
			return err
		}
		return insertBuiltinNodes(tx, BuiltinSkillTreeRev)
	})
}

func insertBuiltinNodes(tx *gorm.DB, treeRev string) error {
	for _, n := range builtinSkillTreeNodes {
		rec := skillTreeNodeRecordFromBuiltin(treeRev, n)
		if err := tx.Create(&rec).Error; err != nil {
			return err
		}
	}
	return nil
}

func ensureBuiltinNodes(tx *gorm.DB, treeRev string) error {
	var count int64
	if err := tx.Model(&models.SkillTreeNodeRecord{}).Where("tree_rev = ?", treeRev).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	return insertBuiltinNodes(tx, treeRev)
}

// SkillTreeNodeRecordFromService exports a service node to a DB record (admin draft clone).
func SkillTreeNodeRecordFromService(treeRev string, n SkillTreeNode) models.SkillTreeNodeRecord {
	return skillTreeNodeRecordFromBuiltin(treeRev, n)
}

func skillTreeNodeRecordFromBuiltin(treeRev string, n SkillTreeNode) models.SkillTreeNodeRecord {
	status := models.SkillTreeNodeStatusActive
	return models.SkillTreeNodeRecord{
		TreeRev:       treeRev,
		Path:          n.Path,
		ParentPath:    n.ParentPath,
		NodeType:      n.NodeType,
		Title:         n.Title,
		Description:   n.Description,
		Topic:         n.Topic,
		SkillKey:      n.SkillKey,
		ProblemKey:    n.ProblemKey,
		CapabilityKey: n.CapabilityKey,
		PackKey:       n.PackKey,
		FeatureKey:    n.FeatureKey,
		ExecutionMode: n.ExecutionMode,
		CLIVisible:    n.CLIVisible,
		Status:        status,
		SortOrder:     n.SortOrder,
		Metadata:      models.NewJSONBFromMap(map[string]interface{}{}),
	}
}

// ActiveSkillTree loads the active tree from DB; on failure returns builtin fallback.
func ActiveSkillTree() ActiveSkillTreeResult {
	_ = SeedBuiltinSkillTree()
	if database.DB == nil {
		return builtinSkillTreeFallback()
	}
	var ver models.SkillTreeVersion
	if err := database.DB.Where("status = ?", models.SkillTreeVersionStatusActive).Order("published_at DESC NULLS LAST, updated_at DESC").First(&ver).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			logger.Warn("ActiveSkillTree: load active version failed, using builtin: %v", err)
		}
		return builtinSkillTreeFallback()
	}
	var rows []models.SkillTreeNodeRecord
	if err := database.DB.Where("tree_rev = ?", ver.TreeRev).Order("sort_order ASC, path ASC").Find(&rows).Error; err != nil {
		logger.Warn("ActiveSkillTree: load nodes failed, using builtin: %v", err)
		return builtinSkillTreeFallback()
	}
	if len(rows) == 0 {
		return builtinSkillTreeFallback()
	}
	nodes := make([]SkillTreeNode, 0, len(rows))
	for _, r := range rows {
		nodes = append(nodes, skillTreeNodeFromRecord(r))
	}
	return ActiveSkillTreeResult{TreeRev: ver.TreeRev, Source: "database", Nodes: nodes}
}

func builtinSkillTreeFallback() ActiveSkillTreeResult {
	return ActiveSkillTreeResult{
		TreeRev: BuiltinSkillTreeRev,
		Source:  "builtin",
		Nodes:   SkillTreeNodesBuiltin(),
	}
}

// SkillTreeNodesBuiltin returns a copy of the embedded builtin tree (fallback only).
func SkillTreeNodesBuiltin() []SkillTreeNode {
	out := make([]SkillTreeNode, len(builtinSkillTreeNodes))
	copy(out, builtinSkillTreeNodes)
	return out
}

func skillTreeNodeFromRecord(r models.SkillTreeNodeRecord) SkillTreeNode {
	return SkillTreeNode{
		Path:          r.Path,
		ParentPath:    r.ParentPath,
		NodeType:      r.NodeType,
		Title:         r.Title,
		Description:   r.Description,
		Topic:         r.Topic,
		SkillKey:      r.SkillKey,
		ProblemKey:    r.ProblemKey,
		CapabilityKey: r.CapabilityKey,
		PackKey:       r.PackKey,
		FeatureKey:    r.FeatureKey,
		ExecutionMode: r.ExecutionMode,
		CLIVisible:    r.CLIVisible,
		SortOrder:     r.SortOrder,
		Status:        r.Status,
	}
}

func activeSkillTreeNodes() []SkillTreeNode {
	return ActiveSkillTree().Nodes
}

func activeSkillTreeRev() string {
	return ActiveSkillTree().TreeRev
}

// SkillTreeNodeByPathActive resolves a node on the active tree.
func SkillTreeNodeByPathActive(path string) (SkillTreeNode, bool) {
	path = strings.TrimSpace(path)
	for _, n := range activeSkillTreeNodes() {
		if n.Path == path {
			return n, true
		}
	}
	return SkillTreeNode{}, false
}

// ListSkillTreeVersions returns all tree revisions for admin.
func ListSkillTreeVersions() ([]models.SkillTreeVersion, error) {
	var rows []models.SkillTreeVersion
	err := database.DB.Order("created_at DESC").Find(&rows).Error
	return rows, err
}

// ValidateSkillTreeNodeInput checks node fields before create/update.
func ValidateSkillTreeNodeInput(treeRev string, path, parentPath, nodeType, skillKey string, excludeID string) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return fmt.Errorf("path 不能为空")
	}
	if strings.Contains(path, "..") {
		return fmt.Errorf("path 非法")
	}
	parentPath = strings.TrimSpace(parentPath)
	if parentPath != "" {
		var count int64
		if err := database.DB.Model(&models.SkillTreeNodeRecord{}).
			Where("tree_rev = ? AND path = ?", treeRev, parentPath).Count(&count).Error; err != nil {
			return err
		}
		if count == 0 {
			return fmt.Errorf("parent_path 不存在")
		}
	}
	switch strings.ToLower(strings.TrimSpace(nodeType)) {
	case SkillNodeTypeCategory, SkillNodeTypeCapability, SkillNodeTypeSkill:
	default:
		return fmt.Errorf("node_type 无效")
	}
	if nodeType == SkillNodeTypeSkill && strings.TrimSpace(skillKey) == "" {
		return fmt.Errorf("skill 节点必须提供 skill_key")
	}
	var dup int64
	q := database.DB.Model(&models.SkillTreeNodeRecord{}).Where("tree_rev = ? AND path = ?", treeRev, path)
	if excludeID != "" {
		q = q.Where("id <> ?", excludeID)
	}
	if err := q.Count(&dup).Error; err != nil {
		return err
	}
	if dup > 0 {
		return fmt.Errorf("path 已存在")
	}
	return nil
}
