package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

func serviceRecoverCmd() *cobra.Command {
	var dryRun, yes, noAI bool
	cmd := &cobra.Command{
		Use:   "recover [service]",
		Short: "基础服务安装失败后分析现场并尝试继续安装",
		Long: `读取 /var/lib/opsfleet/service-deploy/recovery-<service>.json 与本地部署状态，
请求 OpsFleet 恢复计划后执行 allowlist 动作（非 TTY 须 --yes）。

示例:
  sudo ai-sre ops service recover redis
  sudo ai-sre ops service recover mysql --dry-run`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireOpsRoot("服务恢复"); err != nil {
				return err
			}
			if !dryRun {
				if err := requireOpsMutationConfirm(yes, "服务恢复"); err != nil {
					return err
				}
			}
			return runServiceRecover(cmd.Context(), strings.TrimSpace(strings.ToLower(args[0])), serviceRecoverOptions{
				DryRun: dryRun,
				Yes:    yes,
				NoAI:   noAI,
			})
		},
	}
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "只采集现场并展示恢复计划")
	cmd.Flags().BoolVar(&yes, "yes", false, "非 TTY 确认执行恢复动作")
	cmd.Flags().BoolVar(&noAI, "no-ai", false, "跳过服务端分析，仅使用本地规则")
	return cmd
}

type serviceRecoverOptions struct {
	DryRun bool
	Yes    bool
	NoAI   bool
}

func runServiceRecover(ctx context.Context, service string, opts serviceRecoverOptions) error {
	st, _ := loadServiceRecoveryState(service)
	dep, _ := loadServiceDeploymentState(service)
	evidence := collectServiceRecoveryEvidence(service, st, dep)
	var plan installRecoveryPlan
	var err error
	if opts.NoAI {
		plan = localServiceRecoveryPlan(service, evidence)
	} else {
		plan, err = postInstallRecoveryAnalyze(ctx, evidence)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[%s] 服务端恢复分析不可用，回退本地规则: %v\n", progName, err)
			plan = localServiceRecoveryPlan(service, evidence)
		} else {
			postInstallRecoveryEvent(ctx, plan.RequestID, "analyze", "success", plan.Summary)
		}
	}
	return applyServiceRecoveryPlan(ctx, service, dep, evidence, plan, opts)
}

func collectServiceRecoveryEvidence(service string, st *ServiceRecoveryState, dep *serviceDeploymentState) map[string]interface{} {
	ev := map[string]interface{}{
		"topic":        service,
		"operation":    "install_recovery",
		"cli_version":  Version,
		"arch":         goArchToAiSreArch(),
		"api_base":     strings.TrimSpace(resolveOpsfleetAPIBase()),
		"recovery_path": serviceRecoveryStatePath(service),
	}
	if st != nil {
		ev["failed_step"] = st.FailedStep
		ev["exit_code"] = st.ExitCode
		ev["log_tail"] = scrubRecoveryText(st.LogTail)
		ev["last_error"] = scrubRecoveryText(st.LastError)
	}
	if dep != nil {
		ev["deploy_id"] = dep.DeployID
		ev["service_state_present"] = true
	}
	ev["services"] = map[string]string{
		service: systemdActive(service),
	}
	return ev
}

func localServiceRecoveryPlan(service string, ev map[string]interface{}) installRecoveryPlan {
	logTail, _ := ev["log_tail"].(string)
	combined := strings.ToLower(logTail)
	plan := installRecoveryPlan{
		ResumeFrom: "service_install",
		SafeActions: []installRecoveryAction{
			{ID: "resume_service_install", Description: "重新执行 " + service + " 安装", Risk: "medium"},
		},
	}
	switch {
	case strings.Contains(combined, "dpkg frontend lock") || strings.Contains(combined, "unattended-upgr"):
		plan.RootCause = "Ubuntu apt/dpkg 锁被 unattended-upgrades 占用"
		plan.FailedStep = "apt install"
		plan.Summary = "等待 apt 锁释放后重试安装"
		plan.SafeActions = append([]installRecoveryAction{
			{ID: "wait_apt_lock", Description: "等待 dpkg 锁释放（最多 10 分钟）", Risk: "low"},
		}, plan.SafeActions...)
	default:
		plan.RootCause = service + " 安装未完成"
		plan.Summary = "可尝试重新拉取 spec 并执行安装"
	}
	return plan
}

func applyServiceRecoveryPlan(ctx context.Context, service string, dep *serviceDeploymentState, evidence map[string]interface{}, plan installRecoveryPlan, opts serviceRecoverOptions) error {
	MergeExecutionFinishMeta(map[string]interface{}{
		"topic":                   service,
		"operation":               "install_recovery",
		"install_recovery":        true,
		"recovery_root_cause":     plan.RootCause,
		"recovery_failed_step":    plan.FailedStep,
		"recovery_summary":        plan.Summary,
		"recovery_request_id":     plan.RequestID,
		"recovery_need_iteration": plan.NeedIteration,
		"install_recovery_plan":   planToMeta(plan),
		"execution_intent":        buildOpsExecutionIntent([]string{"ops", "service", "recover", service}),
	})
	fmt.Printf("【根因】%s\n", strings.TrimSpace(plan.RootCause))
	if plan.FailedStep != "" {
		fmt.Printf("【失败步骤】%s\n", plan.FailedStep)
	}
	if plan.Summary != "" {
		fmt.Printf("【建议】%s\n", plan.Summary)
	}
	if len(plan.SafeActions) > 0 {
		fmt.Println("【安全动作】")
		for _, a := range plan.SafeActions {
			fmt.Printf("- %s (%s)\n", a.Description, a.ID)
		}
	}
	if opts.DryRun {
		fmt.Println("\n(dry-run: 未执行任何动作)")
		return nil
	}
	for _, action := range plan.SafeActions {
		switch action.ID {
		case "wait_apt_lock":
			if err := waitForAptLock(600); err != nil {
				return err
			}
			appendRecoveryActionMeta(action.ID, "success")
		case "resume_service_install":
			if dep == nil {
				dep, _ = loadServiceDeploymentState(service)
			}
			if dep == nil || dep.APIURL == "" || dep.DeployID == "" || dep.Token == "" {
				return errors.New("未找到服务部署状态，无法继续安装")
			}
			specURL := fmt.Sprintf("%s/api/service-deploy/deployments/%s/spec?token=%s",
				strings.TrimRight(dep.APIURL, "/"), dep.DeployID, dep.Token)
			spec, err := fetchServiceSpec(specURL)
			if err != nil {
				captureServiceInstallFailure(service, "recover", "fetch_spec", 1, err.Error())
				postInstallRecoveryFinish(ctx, plan, "failed", err.Error())
				return err
			}
			report := func(step, status, msg string) {
				fmt.Printf("[%s] %s: %s\n", step, status, msg)
				_ = postServiceEvent(dep.APIURL, dep.DeployID, dep.Token, step, status, msg)
			}
			if err := runServiceTemplate(spec, report); err != nil {
				captureServiceInstallFailure(service, "recover", "install", 1, err.Error())
				postInstallRecoveryFinish(ctx, plan, "failed", err.Error())
				return err
			}
			_ = removeServiceRecoveryState(service)
			postInstallRecoveryFinish(ctx, plan, "success", "service install resumed")
			return nil
		}
	}
	return errors.New("未找到可执行的恢复动作")
}
