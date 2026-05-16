package handlers

import (
	"net/http"
	"strings"
	"time"

	"ft-backend/database"
	"ft-backend/middleware"
	"ft-backend/models"
	"ft-backend/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// GetCLISync returns the compact execution policy snapshot for a bound CLI.
// It deliberately omits YAML, prompts, Stripe details, and review asset content.
func GetCLISync(c *gin.Context) {
	ident, ok := resolveCLIBearerIdentity(c)
	if !ok {
		return
	}
	tree := services.ActiveSkillTree()
	caps := make([]gin.H, 0, len(tree.Nodes))
	for _, n := range tree.Nodes {
		if n.Status == models.SkillTreeNodeStatusDisabled {
			continue
		}
		if !n.CLIVisible || n.NodeType == services.SkillNodeTypeCategory {
			continue
		}
		state := cliCapabilityState(c, ident, n)
		denialReason := ""
		if !state.CanExecute {
			denialReason = state.AccessState
		}
		caps = append(caps, gin.H{
			"node_path":              n.Path,
			"node_type":              n.NodeType,
			"title":                  n.Title,
			"topic":                  n.Topic,
			"skill_key":              n.SkillKey,
			"problem_key":            n.ProblemKey,
			"capability_key":         n.CapabilityKey,
			"pack_key":               n.PackKey,
			"feature_key":            n.FeatureKey,
			"execution_mode":         n.ExecutionMode,
			"can_execute":            state.CanExecute,
			"access_state":           state.AccessState,
			"denial_reason":          denialReason,
			"entitlement_source":     state.EntitlementSource,
			"requires_subscription":  state.RequiresSubscription,
			"requires_plan":          n.ExecutionMode == services.ExecutionModeServerPlanReadonly,
			"local_fallback_allowed": n.ExecutionMode == services.ExecutionModeLocalAIFallback || n.ExecutionMode == services.ExecutionModeLocalReadonly,
		})
	}
	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": gin.H{
			"policy_rev":         time.Now().UTC().Format("2006-01-02"),
			"tree_rev":           tree.TreeRev,
			"tree_source":        tree.Source,
			"min_cli_version":    "0.5.18",
			"latest_cli_version": "0.5.21",
			"capabilities":       caps,
			"ai_quota": gin.H{
				"free_daily_limit": aiFreeDailyLimit,
				"timezone":         "Asia/Shanghai",
			},
		},
		"msg": "success",
	})
}

type cliCapabilityAccess struct {
	CanExecute           bool
	AccessState          string
	EntitlementSource    string
	RequiresSubscription bool
}

func cliCapabilityState(c *gin.Context, ident *aiIdentity, node services.SkillTreeNode) cliCapabilityAccess {
	if ident == nil {
		return cliCapabilityAccess{AccessState: "unauthorized", RequiresSubscription: true}
	}
	if models.IsSuperAdminRole(ident.Role) {
		return cliCapabilityAccess{CanExecute: true, AccessState: "super_admin", EntitlementSource: "super_admin"}
	}
	packKey := strings.TrimSpace(node.PackKey)
	if packKey == "" {
		packKey = skillPackForTopic(node.Topic)
	}
	if strings.HasPrefix(packKey, "skillpack.") {
		return cliAISkillPackAccess(c, ident, packKey)
	}
	featureKey := strings.TrimSpace(node.FeatureKey)
	if featureKey == "" {
		featureKey = firstFeatureForPackage(packKey)
	}
	allowed, payload := middleware.CheckCapability(ident.UserID, ident.Role, featureKey, middleware.CapabilityActionExecute)
	source, _ := payload["entitlement_source"].(string)
	required, _ := payload["billing_required"].(bool)
	if allowed {
		state := "available"
		if source != "" {
			state = source
		}
		return cliCapabilityAccess{CanExecute: true, AccessState: state, EntitlementSource: source, RequiresSubscription: required}
	}
	return cliCapabilityAccess{CanExecute: false, AccessState: "paywall", RequiresSubscription: true}
}

func cliAISkillPackAccess(c *gin.Context, ident *aiIdentity, packKey string) cliCapabilityAccess {
	if ident == nil || ident.UserID == uuid.Nil {
		return cliCapabilityAccess{AccessState: "unauthorized", RequiresSubscription: true}
	}
	if source, ok := cliEntitlementSourceForUser(ident.UserID, packKey); ok {
		if source == "" {
			source = "entitlement"
		}
		return cliCapabilityAccess{CanExecute: true, AccessState: source, EntitlementSource: source}
	}
	used := 0
	var usage models.AIUsage
	if err := databaseLookupAIUsage(ident.Subject, packKey, &usage); err == nil {
		used = usage.Count
	}
	if used < aiFreeDailyLimit {
		return cliCapabilityAccess{CanExecute: true, AccessState: "free_quota", EntitlementSource: "free", RequiresSubscription: true}
	}
	return cliCapabilityAccess{CanExecute: false, AccessState: "paywall", EntitlementSource: "free_exhausted", RequiresSubscription: true}
}

func databaseLookupAIUsage(subject, packKey string, out *models.AIUsage) error {
	return database.DB.Where("subject = ? AND pack_key = ? AND usage_date = ?", subject, packKey, aiQuotaDate()).First(out).Error
}

func cliEntitlementSourceForUser(uid uuid.UUID, packKey string) (string, bool) {
	var ent models.Entitlement
	err := database.DB.Where("user_id = ? AND feature_key = ?", uid, packKey).
		Where("valid_until IS NULL OR valid_until > ?", time.Now().UTC()).
		First(&ent).Error
	if err != nil {
		return "", false
	}
	return strings.TrimSpace(ent.Source), true
}
