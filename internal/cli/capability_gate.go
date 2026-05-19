package cli

import (
	"context"
	"fmt"
	"strings"
)

// ensureExecutionAllowed checks sync capabilities before server plan/AI calls.
func ensureExecutionAllowed(ctx context.Context, intent executionIntent, refresh bool) error {
	return ensureExecutionAllowedWithContext(ctx, intent, refresh, nil)
}

func ensureExecutionAllowedWithContext(ctx context.Context, intent executionIntent, refresh bool, contextKV map[string]string) error {
	if strings.TrimSpace(resolveOpsfleetAPIBase()) == "" {
		return nil
	}
	ensureUpgradeBeforeOpsfleetAPI()
	if err := validateOpsfleetCredentials(); err != nil {
		return err
	}
	err := checkExecutionAllowedFromSync(ctx, intent, refresh)
	if err != nil {
		err = formatOpsfleetAPIError(err, "/api/cli/sync")
	}
	if err == nil {
		return nil
	}
	if !isCapabilityNotFoundError(err) && !isPaywallError(err) {
		return err
	}
	failureKind := "capability_not_found"
	if isPaywallError(err) {
		failureKind = "subscription_required"
	}
	root := newRoot(progName)
	plan, planErr := requestFulfillmentPlan(ctx, root, intent, failureKind, err.Error(), contextKV)
	switch {
	case plan != nil && plan.RetryAllowed && (plan.Action == fulfillmentActionGrantedRetry || plan.Action == fulfillmentActionAutoIterationCreated):
		return checkExecutionAllowedFromSync(ctx, intent, true)
	case plan != nil:
		return fulfillmentPlanUserMessage(plan, err)
	case planErr != nil:
		// Legacy fallback when fulfillment endpoint unavailable.
		if failureKind == "capability_not_found" {
			if gapErr := requestCapabilityGap(ctx, intent, contextKV); gapErr != nil {
				return err
			}
			return checkExecutionAllowedFromSync(ctx, intent, true)
		}
		return err
	default:
		return err
	}
}

func isPaywallError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "能力不可执行") || strings.Contains(err.Error(), "paywall")
}

func checkExecutionAllowedFromSync(ctx context.Context, intent executionIntent, refresh bool) error {
	resp, err := getCLISyncCached(ctx, refresh)
	if err != nil {
		return err
	}
	if resp.UpgradeRequired {
		return fmt.Errorf("当前 CLI 版本过低，请升级至 %s 或以上（本机 %s）", resp.MinCLIVersion, Version)
	}
	cap := findSyncCapability(resp, intent)
	if cap == nil {
		return fmt.Errorf("未在技能树中找到可执行能力（topic=%s problem=%s）", intent.Topic, intent.ProblemKey)
	}
	if cap.CanExecute {
		return nil
	}
	reason := strings.TrimSpace(cap.DenialReason)
	if reason == "" {
		reason = cap.AccessState
	}
	if reason == "" {
		reason = "paywall"
	}
	label := cap.Title
	if label == "" {
		label = cap.NodePath
	}
	return fmt.Errorf("能力不可执行: %s（%s）；请订阅 %s 或联系管理员", label, reason, cap.CommercialProductKey)
}

func findSyncCapability(resp *cliSyncResponse, intent executionIntent) *cliSyncCapability {
	if resp == nil {
		return nil
	}
	path := strings.TrimSpace(intent.NodePath)
	if path == "" {
		path = strings.TrimSpace(intent.CandidateNodePath)
	}
	if path != "" {
		for i := range resp.Capabilities {
			if resp.Capabilities[i].NodePath == path {
				return &resp.Capabilities[i]
			}
		}
	}
	sk := strings.TrimSpace(intent.SkillKey)
	pk := strings.TrimSpace(intent.ProblemKey)
	if sk != "" {
		for i := range resp.Capabilities {
			c := &resp.Capabilities[i]
			if c.SkillKey == sk && (pk == "" || c.ProblemKey == pk) {
				return c
			}
		}
	}
	topic := strings.TrimSpace(intent.Topic)
	if topic != "" {
		for i := range resp.Capabilities {
			c := &resp.Capabilities[i]
			if strings.EqualFold(c.Topic, topic) && (pk == "" || c.ProblemKey == pk) {
				return c
			}
		}
	}
	return nil
}
