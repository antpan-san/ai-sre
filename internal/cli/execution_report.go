package cli

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
)

type executionReporter struct {
	apiBase       string
	inviteID      string
	token         string
	correlationID string
	command       string
	source        string
	category      string
	started       time.Time
}

func newExecutionReporter(programName string, args []string) *executionReporter {
	if os.Getenv("OPSFLEET_EXECUTION_REPORT_DISABLED") == "1" {
		return nil
	}
	apiBase := strings.TrimRight(strings.TrimSpace(os.Getenv("OPSFLEET_API_URL")), "/")
	if apiBase == "" {
		apiBase = strings.TrimRight(strings.TrimSpace(resolveOpsfleetAPIBase()), "/")
	}
	inviteID := ""
	token := strings.TrimSpace(os.Getenv("OPSFLEET_EXECUTION_TOKEN"))
	for _, arg := range args {
		if !strings.HasPrefix(arg, installRefPrefixV1) {
			continue
		}
		if wire, err := decodeInstallRefV1(arg); err == nil {
			apiBase = strings.TrimRight(strings.TrimSpace(wire.B), "/")
			inviteID = wire.I
			token = wire.T
			break
		}
	}
	if token == "" {
		if ref := loadK8sInstallRef(); ref != "" {
			if wire, err := decodeInstallRefV1(ref); err == nil {
				apiBase = strings.TrimRight(strings.TrimSpace(wire.B), "/")
				inviteID = wire.I
				token = wire.T
			}
		}
	}
	if apiBase == "" || token == "" {
		return nil
	}
	cmd := programName
	if len(args) > 0 {
		cmd += " " + strings.Join(args, " ")
	}
	corr := uuid.NewString()
	return &executionReporter{
		apiBase:       apiBase,
		inviteID:      inviteID,
		token:         token,
		correlationID: corr,
		command:       redactExecutionCommand(cmd),
		source:        "cli",
		category:      executionCategory(args),
	}
}

func (r *executionReporter) start() {
	if r == nil {
		return
	}
	r.started = time.Now()
	host, _ := os.Hostname()
	target := executionTargetFromArgv(os.Args[1:])
	if target == "" {
		target = host
	}
	topic := executionTopicFromArgv(os.Args[1:])
	payload := map[string]interface{}{
		"correlation_id":      r.correlationID,
		"source":              r.source,
		"category":            r.category,
		"name":                "ai-sre " + r.category,
		"command":             r.command,
		"target_host":         target,
		"resource_name":       target,
		"resource_type":       r.category,
		"status":              "running",
		"invite_id":           r.inviteID,
		"token":               r.token,
		"rollback_capability": rollbackCapabilityForArgs(os.Args[1:]),
		"rollback_plan":       rollbackPlanForArgs(os.Args[1:]),
		"rollback_advice":     rollbackAdviceForArgs(os.Args[1:]),
		"metadata": map[string]interface{}{
			"record_kind":        "client_execution",
			"argv0":                os.Args[0],
			"version":              Version,
			"topic":                topic,
			"normalized_command":   r.category,
			"hostname":             host,
			"binding_id":           resolveOpsfleetBindingID(),
			"fingerprint_hash":     resolveOpsfleetFingerprint(),
			"diagnosis_target":     target,
		},
	}
	if u := strings.TrimSpace(os.Getenv("OPSFLEET_EXECUTION_USERNAME")); u != "" {
		payload["created_by"] = u
		payload["trigger_user"] = u
	}
	if data := r.post("/api/execution-records/report/start", payload); data != nil {
		id, _ := data["id"].(string)
		if id == "" {
			if nested, ok := data["data"].(map[string]interface{}); ok {
				id, _ = nested["id"].(string)
			}
		}
		setActiveExecution(r.correlationID, id)
	} else {
		setActiveExecution(r.correlationID, "")
	}
}

func (r *executionReporter) finish(err error) {
	if r == nil {
		return
	}
	exitCode := 0
	status := "success"
	stderr := ""
	if err != nil {
		exitCode = 1
		status = "failed"
		stderr = err.Error()
	}
	meta := map[string]interface{}{
		"duration_ms": time.Since(r.started).Milliseconds(),
		"topic":       executionTopicFromArgv(os.Args[1:]),
	}
	for k, v := range drainExecutionFinishMeta() {
		meta[k] = v
	}
	payload := map[string]interface{}{
		"correlation_id": r.correlationID,
		"record_id":      ActiveExecutionRecordID(),
		"invite_id":      r.inviteID,
		"token":          r.token,
		"status":         status,
		"exit_code":      exitCode,
		"stderr_summary": stderr,
		"metadata":       meta,
	}
	if s, ok := meta["summary"].(string); ok && strings.TrimSpace(s) != "" {
		payload["stdout_summary"] = s
	}
	r.post("/api/execution-records/report/finish", payload)
}

func (r *executionReporter) post(path string, payload map[string]interface{}) map[string]interface{} {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil
	}
	client := &http.Client{Timeout: 3 * time.Second}
	req, err := http.NewRequest(http.MethodPost, r.apiBase+path, bytes.NewReader(body))
	if err != nil {
		return nil
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil
	}
	var env struct {
		Code int                    `json:"code"`
		Data map[string]interface{} `json:"data"`
	}
	if json.Unmarshal(raw, &env) != nil || env.Data == nil {
		return nil
	}
	return env.Data
}

func executionTopicFromArgv(args []string) string {
	if len(args) == 0 {
		return ""
	}
	switch args[0] {
	case "check", "analyze", "probe":
		if len(args) >= 2 {
			return strings.ToLower(strings.TrimSpace(args[1]))
		}
	}
	return ""
}

func executionTargetFromArgv(args []string) string {
	if len(args) == 0 {
		return ""
	}
	switch args[0] {
	case "check", "analyze", "probe":
		if len(args) >= 3 {
			return strings.TrimSpace(args[2])
		}
		if len(args) == 2 && checkTopicAcceptsOptionalTarget(args[1]) {
			topic := normalizeCheckTopic(args[1])
			if spec, ok := checkTargetSpecs[topic]; ok {
				return spec.Default
			}
		}
	}
	return ""
}

func executionCategory(args []string) string {
	if len(args) == 0 {
		return "command"
	}
	if args[0] == "ops" && len(args) > 2 {
		switch args[1] {
		case "k8s":
			return "k8s_" + args[2]
		case "service":
			if len(args) > 3 {
				return "service_" + args[2] + "_" + args[3]
			}
		case "uninstall":
			if len(args) > 2 {
				return "ops_uninstall_" + args[2]
			}
			return "ops_uninstall"
		}
	}
	if args[0] == "k8s" && len(args) > 1 {
		return "k8s_" + args[1]
	}
	if args[0] == "job" && len(args) > 1 {
		return "job_" + args[1]
	}
	if args[0] == "node" && len(args) > 2 {
		return "node_" + strings.Join(args[1:3], "_")
	}
	return args[0]
}

func redactExecutionCommand(cmd string) string {
	fields := strings.Fields(cmd)
	for i, f := range fields {
		if strings.HasPrefix(f, installRefPrefixV1) {
			fields[i] = installRefPrefixV1 + "<redacted>"
		}
	}
	return strings.Join(fields, " ")
}

func rollbackCapabilityForArgs(args []string) string {
	if len(args) >= 3 && args[0] == "ops" && args[1] == "k8s" && args[2] == "install" {
		return "auto"
	}
	if len(args) >= 3 && args[0] == "ops" && args[1] == "k8s" && args[2] == "recover" {
		return "manual"
	}
	if len(args) >= 2 && args[0] == "k8s" && args[1] == "install" {
		return "auto"
	}
	if len(args) >= 3 && args[0] == "ops" && args[1] == "k8s" && (args[2] == "cleanup" || args[2] == "uninstall") {
		return "none"
	}
	if len(args) >= 3 && args[0] == "ops" && args[1] == "uninstall" && args[2] == "k8s" {
		return "none"
	}
	if len(args) >= 2 && args[0] == "k8s" && (args[1] == "cleanup" || args[1] == "uninstall") {
		return "none"
	}
	if len(args) >= 3 && args[0] == "node" && args[1] == "tune" {
		return "manual"
	}
	return "none"
}

func rollbackPlanForArgs(args []string) map[string]interface{} {
	if len(args) >= 3 && args[0] == "ops" && args[1] == "k8s" && args[2] == "install" {
		return map[string]interface{}{
			"manual_command": "sudo ai-sre ops k8s recover",
			"cleanup_command": "sudo ai-sre ops uninstall k8s",
		}
	}
	if len(args) >= 2 && args[0] == "k8s" && args[1] == "install" {
		return map[string]interface{}{
			"mode":           "manual_command",
			"command":        "sudo ai-sre uninstall k8s",
			"manual_command": "sudo ai-sre ops k8s cleanup '<same ofpk8s1 ref>'",
		}
	}
	if len(args) >= 3 && args[0] == "node" && args[1] == "tune" {
		return map[string]interface{}{
			"mode":   "manual",
			"advice": "节点调优会修改系统配置，请参考执行记录中的输出与备份文件手动恢复。",
		}
	}
	return map[string]interface{}{}
}

func rollbackAdviceForArgs(args []string) string {
	if len(args) >= 3 && args[0] == "ops" && args[1] == "k8s" && args[2] == "install" {
		return "安装失败时执行 sudo ai-sre ops k8s recover；清理集群 sudo ai-sre ops uninstall k8s。"
	}
	if len(args) >= 3 && args[0] == "ops" && args[1] == "k8s" && args[2] == "recover" {
		return "恢复命令仅执行 allowlist 安全动作；高风险清理请使用 ops uninstall 并显式确认。"
	}
	if len(args) >= 2 && args[0] == "k8s" && args[1] == "install" {
		return "可在控制机执行 sudo ai-sre ops k8s recover；清理: sudo ai-sre ops uninstall k8s；或 sudo ai-sre ops k8s cleanup '<ref>'。"
	}
	if len(args) >= 3 && args[0] == "node" && args[1] == "tune" {
		return "node tune 属于系统配置变更，当前提供人工回滚建议，不自动恢复。"
	}
	return "该命令没有可验证的自动回滚语义。"
}
