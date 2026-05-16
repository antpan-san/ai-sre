package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"ft-backend/database"
	"ft-backend/middleware"
	"ft-backend/models"
	"ft-backend/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	cliMinVersion    = "0.5.18"
	cliLatestVersion = "0.5.22"
)

// GetCLISync returns the compact execution policy snapshot for a bound CLI (sync v2).
// It deliberately omits YAML, prompts, Stripe prices, and full entitlement policies.
func GetCLISync(c *gin.Context) {
	ident, ok := resolveCLIBearerIdentity(c)
	if !ok {
		return
	}
	tree := services.ActiveSkillTree()
	policyRev := services.CommercialPolicyRev()
	templates := map[string]interface{}{}
	caps := make([]gin.H, 0, len(tree.Nodes))
	for _, n := range tree.Nodes {
		if n.Status == models.SkillTreeNodeStatusDisabled {
			continue
		}
		if !n.CLIVisible || n.NodeType == services.SkillNodeTypeCategory {
			continue
		}
		commercialKey := ""
		packKey := strings.TrimSpace(n.PackKey)
		if match, ok := services.ResolveCommercialProductForNode(n.Path); ok {
			commercialKey = match.ProductKey
			if packKey == "" {
				packKey = match.PackKey
			}
		}
		if packKey == "" {
			packKey = skillPackForTopic(n.Topic)
		}
		state := cliCapabilityState(c, ident, n, packKey)
		denialReason := ""
		if !state.CanExecute {
			denialReason = state.AccessState
			if denialReason == "" {
				denialReason = "paywall"
			}
		}
		if topic := strings.TrimSpace(n.Topic); topic != "" {
			if _, ok := templates[topic]; !ok {
				templates[topic] = services.ParameterTemplatesForTopic(topic)
			}
		}
		caps = append(caps, gin.H{
			"node_path":                n.Path,
			"title":                    n.Title,
			"topic":                    n.Topic,
			"skill_key":                n.SkillKey,
			"problem_key":              n.ProblemKey,
			"capability_key":           n.CapabilityKey,
			"pack_key":                 packKey,
			"execution_mode":           n.ExecutionMode,
			"can_execute":              state.CanExecute,
			"access_state":             state.AccessState,
			"denial_reason":            denialReason,
			"commercial_product_key":   commercialKey,
			"requires_plan":            n.ExecutionMode == services.ExecutionModeServerPlanReadonly,
			"local_fallback_allowed":   n.ExecutionMode == services.ExecutionModeLocalAIFallback || n.ExecutionMode == services.ExecutionModeLocalReadonly,
			"entitlement_source":       state.EntitlementSource,
			"requires_subscription":    state.RequiresSubscription,
			"node_type":                n.NodeType,
			"feature_key":              n.FeatureKey,
		})
	}
	cliVer := strings.TrimSpace(c.GetHeader("X-AI-SRE-Version"))
	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": gin.H{
			"policy_rev":         policyRev,
			"tree_rev":           tree.TreeRev,
			"tree_source":        tree.Source,
			"min_cli_version":    cliMinVersion,
			"latest_cli_version": cliLatestVersion,
			"upgrade_required":   cliUpgradeRequired(cliVer),
			"capabilities":       caps,
			"parameter_templates": templates,
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

func cliCapabilityState(c *gin.Context, ident *aiIdentity, node services.SkillTreeNode, packKey string) cliCapabilityAccess {
	if ident == nil {
		return cliCapabilityAccess{AccessState: "unauthorized", RequiresSubscription: true}
	}
	if models.IsSuperAdminRole(ident.Role) {
		return cliCapabilityAccess{CanExecute: true, AccessState: "super_admin", EntitlementSource: "super_admin"}
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
		state := "entitlement"
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
	if source, ok := services.UserHasCommercialEntitlement(ident.UserID, packKey); ok {
		return cliCapabilityAccess{CanExecute: true, AccessState: source, EntitlementSource: source}
	}
	if source, ok := cliEntitlementSourceForUser(ident.UserID, packKey); ok {
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
	return services.UserHasCommercialEntitlement(uid, packKey)
}

func cliUpgradeRequired(clientVersion string) bool {
	if clientVersion == "" {
		return false
	}
	return versionLess(clientVersion, cliMinVersion)
}

func versionLess(a, b string) bool {
	pa := parseVersionParts(a)
	pb := parseVersionParts(b)
	for i := 0; i < 3; i++ {
		if pa[i] < pb[i] {
			return true
		}
		if pa[i] > pb[i] {
			return false
		}
	}
	return false
}

func parseVersionParts(v string) [3]int {
	v = strings.TrimPrefix(strings.TrimSpace(v), "v")
	parts := strings.Split(v, ".")
	var out [3]int
	for i := 0; i < 3 && i < len(parts); i++ {
		out[i], _ = strconv.Atoi(parts[i])
	}
	return out
}
