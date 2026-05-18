package services

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"ft-backend/common/logger"
	"ft-backend/database"
	"ft-backend/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

var (
	commercialSeedOnce sync.Once
)

// SeedSkillCommercialProducts inserts default domain packs and bindings (idempotent).
func SeedSkillCommercialProducts() error {
	var seedErr error
	commercialSeedOnce.Do(func() {
		seedErr = seedSkillCommercialProductsOnce()
	})
	return seedErr
}

func seedSkillCommercialProductsOnce() error {
	if database.DB == nil {
		return nil
	}
	products := []models.SkillCommercialProduct{
		{ProductKey: models.SkillPackK8s, Title: "K8s 诊断技能包", ProductType: models.CommercialProductTypePack, Status: models.CommercialProductStatusActive, SortOrder: 10},
		{ProductKey: models.SkillPackKafka, Title: "Kafka 诊断技能包", ProductType: models.CommercialProductTypePack, Status: models.CommercialProductStatusActive, SortOrder: 20},
		{ProductKey: models.SkillPackRedis, Title: "Redis 诊断技能包", ProductType: models.CommercialProductTypePack, Status: models.CommercialProductStatusActive, SortOrder: 30},
		{ProductKey: models.SkillPackNginx, Title: "Nginx 诊断技能包", ProductType: models.CommercialProductTypePack, Status: models.CommercialProductStatusActive, SortOrder: 40},
		{ProductKey: models.SkillPackMySQL, Title: "MySQL 诊断技能包", ProductType: models.CommercialProductTypePack, Status: models.CommercialProductStatusActive, SortOrder: 50},
		{ProductKey: models.SkillPackPostgreSQL, Title: "PostgreSQL 诊断技能包", ProductType: models.CommercialProductTypePack, Status: models.CommercialProductStatusActive, SortOrder: 55},
		{ProductKey: models.SkillPackElasticsearch, Title: "Elasticsearch 诊断技能包", ProductType: models.CommercialProductTypePack, Status: models.CommercialProductStatusActive, SortOrder: 60},
		{ProductKey: models.PackKeyK8sDelivery, Title: "K8s 交付实施", ProductType: models.CommercialProductTypePack, Status: models.CommercialProductStatusActive, SortOrder: 110},
		{ProductKey: models.PackKeyRuntimeObserve, Title: "Go 运行时观测", ProductType: models.CommercialProductTypePack, Status: models.CommercialProductStatusActive, SortOrder: 120},
	}
	for _, p := range products {
		var existing models.SkillCommercialProduct
		err := database.DB.Where("product_key = ?", p.ProductKey).First(&existing).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			if err := database.DB.Create(&p).Error; err != nil {
				return err
			}
		} else if err != nil {
			return err
		}
	}
	bindings := defaultCommercialBindings()
	for _, b := range bindings {
		var count int64
		if err := database.DB.Model(&models.SkillProductNodeBinding{}).
			Where("product_key = ? AND node_path = ? AND grant_scope = ?", b.ProductKey, b.NodePath, b.GrantScope).
			Count(&count).Error; err != nil {
			return err
		}
		if count == 0 {
			if err := database.DB.Create(&b).Error; err != nil {
				return err
			}
		}
	}
	return nil
}

func defaultCommercialBindings() []models.SkillProductNodeBinding {
	return []models.SkillProductNodeBinding{
		{ProductKey: models.SkillPackK8s, NodePath: "ops.incident_diagnosis.kubernetes", PackKey: models.SkillPackK8s, GrantScope: models.ProductGrantScopeSubtree},
		{ProductKey: models.SkillPackKafka, NodePath: "ops.incident_diagnosis.middleware.kafka", PackKey: models.SkillPackKafka, GrantScope: models.ProductGrantScopeSubtree},
		{ProductKey: models.SkillPackRedis, NodePath: "ops.incident_diagnosis.middleware.redis", PackKey: models.SkillPackRedis, GrantScope: models.ProductGrantScopeSubtree},
		{ProductKey: models.SkillPackNginx, NodePath: "ops.incident_diagnosis.middleware.nginx", PackKey: models.SkillPackNginx, GrantScope: models.ProductGrantScopeSubtree},
		{ProductKey: models.SkillPackMySQL, NodePath: "ops.incident_diagnosis.middleware.mysql", PackKey: models.SkillPackMySQL, GrantScope: models.ProductGrantScopeSubtree},
		{ProductKey: models.SkillPackPostgreSQL, NodePath: "ops.incident_diagnosis.middleware.postgresql", PackKey: models.SkillPackPostgreSQL, GrantScope: models.ProductGrantScopeSubtree},
		{ProductKey: models.SkillPackElasticsearch, NodePath: "ops.incident_diagnosis.middleware.elasticsearch", PackKey: models.SkillPackElasticsearch, GrantScope: models.ProductGrantScopeSubtree},
		{ProductKey: models.PackKeyK8sDelivery, NodePath: "ops.delivery_implementation.kubernetes", PackKey: models.PackKeyK8sDelivery, GrantScope: models.ProductGrantScopeSubtree},
		{ProductKey: models.PackKeyRuntimeObserve, NodePath: "ops.incident_diagnosis.application.go_runtime", PackKey: models.PackKeyRuntimeObserve, GrantScope: models.ProductGrantScopeSubtree},
	}
}

// CommercialPolicyRev returns a stable revision for commercial bindings (CLI cache invalidation).
func CommercialPolicyRev() string {
	if database.DB == nil {
		return "builtin.commercial.v1"
	}
	var products []models.SkillCommercialProduct
	_ = database.DB.Select("product_key", "updated_at").Order("product_key").Find(&products).Error
	var bindings []models.SkillProductNodeBinding
	_ = database.DB.Select("product_key", "node_path", "grant_scope", "updated_at").Order("product_key, node_path").Find(&bindings).Error
	h := sha256.New()
	for _, p := range products {
		h.Write([]byte(p.ProductKey))
		h.Write([]byte(p.UpdatedAt.UTC().Format(time.RFC3339Nano)))
	}
	for _, b := range bindings {
		h.Write([]byte(b.ProductKey))
		h.Write([]byte(b.NodePath))
		h.Write([]byte(b.GrantScope))
		h.Write([]byte(b.UpdatedAt.UTC().Format(time.RFC3339Nano)))
	}
	sum := hex.EncodeToString(h.Sum(nil))
	if len(sum) > 16 {
		sum = sum[:16]
	}
	return "commercial." + sum
}

// CommercialProductMatch describes how a tree node maps to a sellable product.
type CommercialProductMatch struct {
	ProductKey string
	PackKey    string
	GrantScope string
}

// ResolveCommercialProductForNode finds the best matching product for a node path.
func ResolveCommercialProductForNode(nodePath string) (CommercialProductMatch, bool) {
	nodePath = strings.TrimSpace(nodePath)
	if nodePath == "" {
		return CommercialProductMatch{}, false
	}
	var bindings []models.SkillProductNodeBinding
	if database.DB == nil {
		return fallbackCommercialMatch(nodePath)
	}
	if err := database.DB.Find(&bindings).Error; err != nil || len(bindings) == 0 {
		return fallbackCommercialMatch(nodePath)
	}
	bestLen := -1
	var best CommercialProductMatch
	for _, b := range bindings {
		if !bindingCoversPath(b, nodePath) {
			continue
		}
		pk := strings.TrimSpace(b.PackKey)
		if pk == "" {
			pk = b.ProductKey
		}
		plen := len(strings.TrimSpace(b.NodePath))
		if plen > bestLen {
			bestLen = plen
			best = CommercialProductMatch{ProductKey: b.ProductKey, PackKey: pk, GrantScope: b.GrantScope}
		}
	}
	if bestLen >= 0 {
		return best, true
	}
	return fallbackCommercialMatch(nodePath)
}

func bindingCoversPath(b models.SkillProductNodeBinding, nodePath string) bool {
	base := strings.TrimSpace(b.NodePath)
	if base == "" {
		return false
	}
	switch strings.TrimSpace(b.GrantScope) {
	case models.ProductGrantScopeNode:
		return nodePath == base
	case models.ProductGrantScopePack:
		return true
	default:
		return nodePath == base || strings.HasPrefix(nodePath, base+".")
	}
}

func fallbackCommercialMatch(nodePath string) (CommercialProductMatch, bool) {
	for _, n := range builtinSkillTreeNodes {
		if n.Path != nodePath {
			continue
		}
		pk := strings.TrimSpace(n.PackKey)
		if pk == "" {
			return CommercialProductMatch{}, false
		}
		productKey := pk
		if strings.HasPrefix(pk, "skillpack.") || strings.HasPrefix(pk, "pack.") {
			productKey = pk
		}
		return CommercialProductMatch{ProductKey: productKey, PackKey: pk, GrantScope: models.ProductGrantScopeSubtree}, true
	}
	return CommercialProductMatch{}, false
}

// ListCommercialProducts returns active products for admin.
func ListCommercialProducts() ([]models.SkillCommercialProduct, error) {
	var rows []models.SkillCommercialProduct
	err := database.DB.Where("status = ?", models.CommercialProductStatusActive).Order("sort_order ASC, product_key ASC").Find(&rows).Error
	return rows, err
}

// ListProductBindings returns bindings, optionally filtered by product_key.
func ListProductBindings(productKey string) ([]models.SkillProductNodeBinding, error) {
	q := database.DB.Order("product_key ASC, node_path ASC")
	if productKey != "" {
		q = q.Where("product_key = ?", productKey)
	}
	var rows []models.SkillProductNodeBinding
	return rows, q.Find(&rows).Error
}

// ProductsForNodePath lists product keys that authorize a node (for admin UI).
func ProductsForNodePath(nodePath string) []string {
	var out []string
	seen := map[string]struct{}{}
	var bindings []models.SkillProductNodeBinding
	if database.DB != nil {
		_ = database.DB.Find(&bindings).Error
	}
	for _, b := range bindings {
		if !bindingCoversPath(b, nodePath) {
			continue
		}
		if _, ok := seen[b.ProductKey]; ok {
			continue
		}
		seen[b.ProductKey] = struct{}{}
		out = append(out, b.ProductKey)
	}
	sort.Strings(out)
	return out
}

// UserHasCommercialEntitlement checks pack_key / product_key entitlement (not quota).
func UserHasCommercialEntitlement(userID uuid.UUID, packOrProductKey string) (string, bool) {
	if userID == uuid.Nil || packOrProductKey == "" {
		return "", false
	}
	var ent models.Entitlement
	err := database.DB.Where("user_id = ? AND feature_key = ?", userID, packOrProductKey).
		Where("valid_until IS NULL OR valid_until > ?", time.Now().UTC()).
		First(&ent).Error
	if err != nil {
		return "", false
	}
	src := strings.TrimSpace(ent.Source)
	if src == "" {
		src = "entitlement"
	}
	return src, true
}

// CreateProductBinding adds a binding after validation.
func CreateProductBinding(b models.SkillProductNodeBinding) error {
	b.ProductKey = strings.TrimSpace(b.ProductKey)
	b.NodePath = strings.TrimSpace(b.NodePath)
	if b.ProductKey == "" || b.NodePath == "" {
		return fmt.Errorf("product_key 与 node_path 必填")
	}
	if b.GrantScope == "" {
		b.GrantScope = models.ProductGrantScopeSubtree
	}
	var prod models.SkillCommercialProduct
	if err := database.DB.Where("product_key = ?", b.ProductKey).First(&prod).Error; err != nil {
		return fmt.Errorf("商品包不存在")
	}
	return database.DB.Create(&b).Error
}

// DeleteProductBinding removes a binding row (bindings are relational, not tree nodes).
func DeleteProductBinding(id uuid.UUID) error {
	return database.DB.Where("id = ?", id).Delete(&models.SkillProductNodeBinding{}).Error
}

// ParameterTemplatesForTopic returns safe CLI parameter hints (no secrets).
func ParameterTemplatesForTopic(topic string) map[string]interface{} {
	topic = strings.ToLower(strings.TrimSpace(topic))
	switch topic {
	case "k8s", "kubernetes":
		return map[string]interface{}{
			"namespace": map[string]string{"type": "string", "example": "prod"},
			"pod":       map[string]string{"type": "string", "example": "api-0"},
			"issue":     map[string]string{"type": "string", "example": "pending"},
		}
	case "go_runtime", "go-runtime":
		return map[string]interface{}{
			"pod":        map[string]string{"type": "string", "example": "prod/api-0"},
			"deployment": map[string]string{"type": "string", "example": "prod/api"},
		}
	case "kafka":
		return map[string]interface{}{"lag": map[string]string{"type": "string", "example": "9000"}}
	default:
		return map[string]interface{}{}
	}
}

// LogCommercialSeedFailure is used from main when seed fails.
func LogCommercialSeedFailure(err error) {
	if err != nil {
		logger.Warn("SeedSkillCommercialProducts: %v", err)
	}
}
