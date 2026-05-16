package cli

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

// ensureExecutionAllowed checks sync capabilities before server plan/AI calls.
func ensureExecutionAllowed(ctx context.Context, intent executionIntent, refresh bool) error {
	if strings.TrimSpace(resolveOpsfleetAPIBase()) == "" {
		return nil
	}
	if strings.TrimSpace(resolveOpsfleetToken()) == "" {
		return errors.New("需要绑定 OpsFleet CLI token；请从控制台安装 ai-sre 后重试")
	}
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
