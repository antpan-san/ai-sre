package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

type installRecoveryPlan struct {
	RootCause    string                   `json:"root_cause"`
	FailedStep   string                   `json:"failed_step"`
	Summary      string                   `json:"summary"`
	SafeActions  []installRecoveryAction  `json:"safe_actions"`
	ResumeFrom   string                   `json:"resume_from"`
	NeedIteration bool                    `json:"need_iteration"`
	RequestID    string                   `json:"request_id,omitempty"`
}

type installRecoveryAction struct {
	ID          string `json:"id"`
	Description string `json:"description"`
	Risk        string `json:"risk,omitempty"`
}

func k8sRecoverCmd() *cobra.Command {
	var (
		dryRun       bool
		yes          bool
		noAI         bool
		cleanupFirst bool
	)
	cmd := &cobra.Command{
		Use:   "recover [install-ref]",
		Short: "K8s 安装失败后分析现场、执行安全恢复动作并可选继续安装",
		Long: `读取 ` + K8sRecoveryStatePath + ` 与 ` + K8sLastBundlePath + `，
采集失败步骤、日志尾部、inventory、SSH/端口/磁盘/containerd/kubelet 状态，
请求 OpsFleet 恢复计划后仅执行本地 allowlist 动作（非 TTY 须 --yes）。

示例:
  sudo ai-sre ops k8s recover
  sudo ai-sre ops k8s recover 'ofpk8s1.…'
  sudo ai-sre ops k8s recover --dry-run
  sudo ai-sre ops k8s recover --cleanup-first --yes`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireOpsRoot("K8s 恢复"); err != nil {
				return err
			}
			if !dryRun {
				if err := requireOpsMutationConfirm(yes, "K8s 恢复"); err != nil {
					return err
				}
			}
			ref := ""
			if len(args) == 1 {
				ref = strings.Trim(strings.TrimSpace(args[0]), `"'`)
			}
			return runK8sRecover(cmd.Context(), k8sRecoverOptions{
				InstallRef:   ref,
				DryRun:       dryRun,
				Yes:          yes,
				NoAI:         noAI,
				CleanupFirst: cleanupFirst,
			})
		},
	}
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "只采集现场并展示恢复计划，不执行动作")
	cmd.Flags().BoolVar(&yes, "yes", false, "非 TTY 环境确认执行恢复动作")
	cmd.Flags().BoolVar(&noAI, "no-ai", false, "跳过服务端 AI 分析，仅使用本地规则")
	cmd.Flags().BoolVar(&cleanupFirst, "cleanup-first", false, "恢复前先对 inventory 节点执行 pre_cleanup")
	return cmd
}

type k8sRecoverOptions struct {
	InstallRef   string
	DryRun       bool
	Yes          bool
	NoAI         bool
	CleanupFirst bool
}

func runK8sRecover(ctx context.Context, opts k8sRecoverOptions) error {
	st, _ := loadK8sRecoveryState()
	if strings.TrimSpace(opts.InstallRef) != "" {
		saveK8sInstallRef(opts.InstallRef)
		if st == nil {
			st = &K8sRecoveryState{InstallRef: opts.InstallRef}
		} else {
			st.InstallRef = opts.InstallRef
		}
	}
	evidence := collectK8sRecoveryEvidence(st)
	if opts.NoAI {
		plan := localInstallRecoveryPlan(evidence)
		return applyK8sRecoveryPlan(ctx, evidence, plan, opts)
	}
	plan, err := postInstallRecoveryAnalyze(ctx, evidence)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[%s] 服务端恢复分析不可用，回退本地规则: %v\n", progName, err)
		plan = localInstallRecoveryPlan(evidence)
	} else {
		postInstallRecoveryEvent(ctx, plan.RequestID, "analyze", "success", plan.Summary)
	}
	return applyK8sRecoveryPlan(ctx, evidence, plan, opts)
}

func collectK8sRecoveryEvidence(st *K8sRecoveryState) map[string]interface{} {
	ev := map[string]interface{}{
		"topic":        "k8s",
		"operation":    "install_recovery",
		"cli_version":  Version,
		"arch":         goArchToAiSreArch(),
		"api_base":     strings.TrimSpace(resolveOpsfleetAPIBase()),
		"install_ref":  redactInstallRefForUpload(st),
		"last_bundle":  K8sLastBundlePath,
		"recovery_path": K8sRecoveryStatePath,
	}
	if st != nil {
		ev["failed_step"] = st.FailedStep
		ev["exit_code"] = st.ExitCode
		ev["log_tail"] = scrubRecoveryText(st.LogTail)
		ev["bundle_root"] = st.BundleRoot
	}
	root := mergeRecoveryBundleRoot(st)
	if root != "" {
		ev["bundle_root"] = root
		ev["inventory_exists"] = fileExists(filepath.Join(root, "inventory", "hosts.ini"))
		ev["install_sh_executable"] = isExecutable(filepath.Join(root, "install.sh"))
		if b, err := os.ReadFile(filepath.Join(root, ".opsfleet-k8s-state")); err == nil {
			ev["state_tail"] = scrubRecoveryText(string(b))
		}
		ev["ssh_preflight"] = runSSHInventoryPreflight(root)
	}
	ev["disk_root_pct"] = diskUsePct("/")
	ev["services"] = map[string]string{
		"containerd": systemdActive("containerd"),
		"kubelet":    systemdActive("kubelet"),
	}
	if ref := loadK8sInstallRef(); ref != "" {
		ev["install_ref_present"] = true
	}
	return ev
}

func redactInstallRefForUpload(st *K8sRecoveryState) string {
	ref := ""
	if st != nil {
		ref = strings.TrimSpace(st.InstallRef)
	}
	if ref == "" {
		ref = loadK8sInstallRef()
	}
	if ref == "" {
		return ""
	}
	if wire, err := decodeInstallRefV1(ref); err == nil {
		return installRefPrefixV1 + wire.I + "…"
	}
	return installRefPrefixV1 + "…"
}

func scrubRecoveryText(s string) string {
	s = strings.TrimSpace(s)
	if len(s) > 6000 {
		s = s[len(s)-6000:]
	}
	for _, secret := range []string{"password", "token", "secret", "Authorization"} {
		if strings.Contains(strings.ToLower(s), strings.ToLower(secret)) {
			s = "[redacted log tail]"
			break
		}
	}
	return s
}

func fileExists(p string) bool {
	st, err := os.Stat(p)
	return err == nil && !st.IsDir()
}

func isExecutable(p string) bool {
	st, err := os.Stat(p)
	if err != nil || st.IsDir() {
		return false
	}
	return st.Mode()&0o111 != 0
}

func diskUsePct(mount string) string {
	out, err := exec.Command("df", "-P", mount).CombinedOutput()
	if err != nil {
		return ""
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) < 2 {
		return ""
	}
	fields := strings.Fields(lines[1])
	if len(fields) < 5 {
		return ""
	}
	return strings.TrimSuffix(fields[4], "%")
}

func systemdActive(unit string) string {
	out, err := exec.Command("systemctl", "is-active", unit).CombinedOutput()
	if err != nil {
		return strings.TrimSpace(string(out))
	}
	return strings.TrimSpace(string(out))
}

func runSSHInventoryPreflight(bundleRoot string) map[string]interface{} {
	inv := filepath.Join(bundleRoot, "inventory", "hosts.ini")
	if !fileExists(inv) {
		return map[string]interface{}{"status": "missing_inventory"}
	}
	out, err := exec.Command("ansible", "all", "-i", inv, "-m", "ping", "-o", "--one-line").CombinedOutput()
	summary := strings.TrimSpace(string(out))
	if len(summary) > 2000 {
		summary = summary[len(summary)-2000:]
	}
	status := "ok"
	if err != nil {
		status = "fail"
	}
	return map[string]interface{}{"status": status, "output_tail": summary}
}

func localInstallRecoveryPlan(ev map[string]interface{}) installRecoveryPlan {
	logTail, _ := ev["log_tail"].(string)
	stateTail, _ := ev["state_tail"].(string)
	combined := strings.ToLower(logTail + "\n" + stateTail)
	plan := installRecoveryPlan{
		RequestID: uuid.NewString(),
		ResumeFrom: "install.sh",
		SafeActions: []installRecoveryAction{},
	}
	switch {
	case strings.Contains(combined, "dpkg frontend lock") || strings.Contains(combined, "unattended-upgr"):
		plan.RootCause = "Ubuntu apt/dpkg 锁被 unattended-upgrades 占用"
		plan.FailedStep = "kube_proxy dependencies / apt install"
		plan.Summary = "等待后台 apt 完成后重试安装；或先执行 pre_cleanup 再 recover"
		plan.SafeActions = append(plan.SafeActions, installRecoveryAction{ID: "wait_apt_lock", Description: "等待 dpkg 锁释放（最多 10 分钟）", Risk: "low"})
	case strings.Contains(combined, "permission denied") && strings.Contains(combined, "install.sh"):
		plan.RootCause = "install.sh 或脚本无执行权限"
		plan.FailedStep = "install.sh"
		plan.SafeActions = append(plan.SafeActions, installRecoveryAction{ID: "chmod_install_scripts", Description: "为 install.sh 与 ansible 脚本添加执行权限", Risk: "low"})
	case strings.Contains(combined, "ssh") && (strings.Contains(combined, "permission denied") || strings.Contains(combined, "host key")):
		plan.RootCause = "SSH 免密或 host key 预检失败"
		plan.FailedStep = "ssh preflight"
		plan.Summary = "请修复 inventory 中各节点 root 免密后重试 recover"
	default:
		plan.RootCause = "K8s 离线安装未完成"
		plan.Summary = "可尝试修复脚本权限后继续 install.sh"
	}
	if ssh, ok := ev["ssh_preflight"].(map[string]interface{}); ok && ssh["status"] == "fail" {
		plan.RootCause = "Ansible SSH 预检失败"
		plan.FailedStep = "ansible ping"
	}
	if execOK, ok := ev["install_sh_executable"].(bool); ok && !execOK {
		plan.SafeActions = append(plan.SafeActions, installRecoveryAction{ID: "chmod_install_scripts", Description: "修复 install.sh 权限", Risk: "low"})
	}
	plan.SafeActions = append(plan.SafeActions, installRecoveryAction{ID: "resume_install", Description: "从 last-bundle 继续执行 install.sh", Risk: "medium"})
	return plan
}

func postInstallRecoveryAnalyze(ctx context.Context, evidence map[string]interface{}) (installRecoveryPlan, error) {
	var plan installRecoveryPlan
	base := strings.TrimRight(strings.TrimSpace(resolveOpsfleetAPIBase()), "/")
	tok := strings.TrimSpace(resolveOpsfleetToken())
	if base == "" || tok == "" {
		return plan, errors.New("missing OPSFLEET_API_URL or CLI token")
	}
	body, _ := json.Marshal(map[string]interface{}{
		"topic":     "k8s",
		"operation": "install_recovery",
		"context":   evidence,
		"command":   strings.Join(os.Args, " "),
		"request_id": uuid.NewString(),
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/api/cli/install-recovery/analyze", bytes.NewReader(body))
	if err != nil {
		return plan, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("X-Opsfleet-Fingerprint", resolveOpsfleetFingerprint())
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return plan, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return plan, fmt.Errorf("HTTP %d: %s", resp.StatusCode, truncateForErr(raw, 512))
	}
	var env struct {
		Data installRecoveryPlan `json:"data"`
	}
	if err := json.Unmarshal(raw, &env); err != nil {
		return plan, err
	}
	if env.Data.RootCause == "" && env.Data.Summary == "" {
		return plan, errors.New("empty recovery plan")
	}
	return env.Data, nil
}

func postInstallRecoveryEvent(ctx context.Context, requestID, step, status, message string) {
	base := strings.TrimRight(strings.TrimSpace(resolveOpsfleetAPIBase()), "/")
	tok := strings.TrimSpace(resolveOpsfleetToken())
	if base == "" || tok == "" || requestID == "" {
		return
	}
	body, _ := json.Marshal(map[string]string{
		"request_id": requestID,
		"step":       step,
		"status":     status,
		"message":    message,
	})
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, base+"/api/cli/install-recovery/events", bytes.NewReader(body))
	if req == nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tok)
	c, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	req = req.WithContext(c)
	resp, err := http.DefaultClient.Do(req)
	if err == nil {
		resp.Body.Close()
	}
}

func applyK8sRecoveryPlan(ctx context.Context, evidence map[string]interface{}, plan installRecoveryPlan, opts k8sRecoverOptions) error {
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
	root, _ := evidence["bundle_root"].(string)
	if strings.TrimSpace(root) == "" {
		root = mergeRecoveryBundleRoot(nil)
	}
	if opts.CleanupFirst && root != "" {
		if err := runCleanupPlaybook(filepath.Join(root, "ansible-agent"), filepath.Join(root, "inventory", "hosts.ini")); err != nil {
			return fmt.Errorf("pre_cleanup 失败: %w", err)
		}
		postInstallRecoveryEvent(ctx, plan.RequestID, "cleanup_first", "success", "pre_cleanup completed")
	}
	for _, action := range plan.SafeActions {
		switch action.ID {
		case "wait_apt_lock":
			if err := waitForAptLock(600); err != nil {
				return err
			}
		case "chmod_install_scripts":
			if root != "" {
				_ = exec.Command("chmod", "+x", filepath.Join(root, "install.sh")).Run()
			}
		case "resume_install":
			if root == "" {
				return errors.New("未找到 last-bundle，无法继续 install.sh")
			}
			if err := runInstallSh(root); err != nil {
				captureK8sInstallFailure(root, loadK8sInstallRef(), "recover", 1, plan.ResumeFrom)
				postInstallRecoveryFinish(ctx, plan, "failed", err.Error())
				return err
			}
			postInstallRecoveryFinish(ctx, plan, "success", "install resumed")
			return nil
		}
	}
	if root != "" && fileExists(filepath.Join(root, "install.sh")) {
		return runInstallSh(root)
	}
	return errors.New("未找到可执行的恢复动作或 last-bundle")
}

func waitForAptLock(maxSec int) error {
	deadline := time.Now().Add(time.Duration(maxSec) * time.Second)
	for time.Now().Before(deadline) {
		if !aptLockHeld() {
			return nil
		}
		time.Sleep(5 * time.Second)
	}
	return errors.New("等待 apt/dpkg 锁超时")
}

func aptLockHeld() bool {
	for _, lock := range []string{"/var/lib/dpkg/lock-frontend", "/var/lib/dpkg/lock"} {
		if fileExists(lock) {
			if out, _ := exec.Command("fuser", lock).CombinedOutput(); len(strings.TrimSpace(string(out))) > 0 {
				return true
			}
		}
	}
	out, _ := exec.Command("pgrep", "-x", "unattended-upgr").CombinedOutput()
	return len(strings.TrimSpace(string(out))) > 0
}

func postInstallRecoveryFinish(ctx context.Context, plan installRecoveryPlan, status, message string) {
	base := strings.TrimRight(strings.TrimSpace(resolveOpsfleetAPIBase()), "/")
	tok := strings.TrimSpace(resolveOpsfleetToken())
	if base == "" || tok == "" {
		return
	}
	body, _ := json.Marshal(map[string]interface{}{
		"request_id":     plan.RequestID,
		"status":         status,
		"message":        message,
		"need_iteration": plan.NeedIteration,
		"root_cause":     plan.RootCause,
	})
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, base+"/api/cli/install-recovery/finish", bytes.NewReader(body))
	if req == nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tok)
	resp, err := http.DefaultClient.Do(req)
	if err == nil {
		resp.Body.Close()
	}
}
