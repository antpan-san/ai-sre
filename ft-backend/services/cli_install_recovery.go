package services

import (
	"strings"

	"github.com/google/uuid"
)

// InstallRecoveryAction is one allowlisted recovery step the CLI may execute locally.
type InstallRecoveryAction struct {
	ID          string `json:"id"`
	Description string `json:"description"`
	Risk        string `json:"risk,omitempty"`
}

// InstallRecoveryPlan is returned to CLI for structured install failure recovery.
type InstallRecoveryPlan struct {
	RequestID     string                  `json:"request_id"`
	RootCause     string                  `json:"root_cause"`
	FailedStep    string                  `json:"failed_step"`
	Summary       string                  `json:"summary"`
	SafeActions   []InstallRecoveryAction `json:"safe_actions"`
	ResumeFrom    string                  `json:"resume_from"`
	NeedIteration bool                    `json:"need_iteration"`
}

// AnalyzeInstallRecovery builds a recovery plan from CLI-collected evidence (rule-first, optional AI later).
func AnalyzeInstallRecovery(topic, operation, command string, ctx map[string]interface{}) InstallRecoveryPlan {
	plan := InstallRecoveryPlan{
		RequestID:  uuid.NewString(),
		ResumeFrom: "install.sh",
	}
	logTail := strings.ToLower(strCtx(ctx, "log_tail") + "\n" + strCtx(ctx, "state_tail"))
	switch {
	case strings.Contains(logTail, "dpkg frontend lock") || strings.Contains(logTail, "unattended-upgr"):
		plan.RootCause = "Ubuntu apt/dpkg 锁被 unattended-upgrades 占用"
		plan.FailedStep = "apt dependencies"
		plan.Summary = "等待后台 apt 完成后 recover；必要时 --cleanup-first"
		plan.SafeActions = []InstallRecoveryAction{
			{ID: "wait_apt_lock", Description: "等待 dpkg 锁释放", Risk: "low"},
			{ID: "resume_install", Description: "继续 install.sh", Risk: "medium"},
		}
		plan.NeedIteration = true
	case strings.Contains(logTail, "permission denied") && strings.Contains(logTail, "install.sh"):
		plan.RootCause = "install.sh 无执行权限"
		plan.FailedStep = "install.sh"
		plan.SafeActions = []InstallRecoveryAction{
			{ID: "chmod_install_scripts", Description: "修复 install.sh 权限", Risk: "low"},
			{ID: "resume_install", Description: "继续 install.sh", Risk: "medium"},
		}
	default:
		plan.RootCause = "K8s 离线安装失败"
		plan.Summary = "采集现场后尝试修复脚本权限并继续 install.sh"
		plan.SafeActions = []InstallRecoveryAction{
			{ID: "chmod_install_scripts", Description: "修复 install.sh 权限", Risk: "low"},
			{ID: "resume_install", Description: "继续 install.sh", Risk: "medium"},
		}
	}
	if ssh, ok := ctx["ssh_preflight"].(map[string]interface{}); ok && strCtx(ssh, "status") == "fail" {
		plan.RootCause = "Ansible SSH 预检失败"
		plan.FailedStep = "ansible ping"
		plan.Summary = "修复 inventory 节点 root 免密 SSH 后重试 recover"
		plan.NeedIteration = true
	}
	if topic == "" {
		topic = "k8s"
	}
	_ = operation
	_ = command
	return plan
}

func strCtx(m map[string]interface{}, key string) string {
	if m == nil {
		return ""
	}
	v, _ := m[key].(string)
	return strings.TrimSpace(v)
}

// FinishInstallRecovery may trigger feedback/auto-iteration when recovery indicates platform gaps.
func FinishInstallRecovery(userID uuid.UUID, requestID, status, rootCause string, needIteration bool) {
	if !needIteration || status == "success" {
		return
	}
	_, _ = AnalyzeCLIFeedback(userID, nil, "k8s", "ops k8s recover", rootCause, map[string]interface{}{
		"request_id":  requestID,
		"classification": "install_recovery_gap",
		"need_iteration": true,
	})
}
