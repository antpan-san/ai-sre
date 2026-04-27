package cli

import (
	"bytes"
	"encoding/json"
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
	if apiBase == "" || token == "" {
		return nil
	}
	cmd := programName
	if len(args) > 0 {
		cmd += " " + strings.Join(args, " ")
	}
	return &executionReporter{
		apiBase:       apiBase,
		inviteID:      inviteID,
		token:         token,
		correlationID: uuid.NewString(),
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
	payload := map[string]interface{}{
		"correlation_id":      r.correlationID,
		"source":              r.source,
		"category":            r.category,
		"name":                "ai-sre " + r.category,
		"command":             r.command,
		"target_host":         host,
		"status":              "running",
		"invite_id":           r.inviteID,
		"token":               r.token,
		"rollback_capability": rollbackCapabilityForArgs(os.Args[1:]),
		"rollback_plan":       rollbackPlanForArgs(os.Args[1:]),
		"rollback_advice":     rollbackAdviceForArgs(os.Args[1:]),
		"metadata": map[string]interface{}{
			"argv0":   os.Args[0],
			"version": Version,
		},
	}
	r.post("/api/execution-records/report/start", payload)
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
	payload := map[string]interface{}{
		"correlation_id": r.correlationID,
		"invite_id":      r.inviteID,
		"token":          r.token,
		"status":         status,
		"exit_code":      exitCode,
		"stderr_summary": stderr,
		"metadata": map[string]interface{}{
			"duration_ms": time.Since(r.started).Milliseconds(),
		},
	}
	r.post("/api/execution-records/report/finish", payload)
}

func (r *executionReporter) post(path string, payload map[string]interface{}) {
	body, err := json.Marshal(payload)
	if err != nil {
		return
	}
	client := &http.Client{Timeout: 2 * time.Second}
	req, err := http.NewRequest(http.MethodPost, r.apiBase+path, bytes.NewReader(body))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return
	}
	_ = resp.Body.Close()
}

func executionCategory(args []string) string {
	if len(args) == 0 {
		return "command"
	}
	if args[0] == "k8s" && len(args) > 1 {
		return "k8s_" + args[1]
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
	if len(args) >= 2 && args[0] == "k8s" && args[1] == "install" {
		return "auto"
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
	if len(args) >= 2 && args[0] == "k8s" && args[1] == "install" {
		return map[string]interface{}{
			"mode":           "manual_command",
			"command":        "sudo ai-sre uninstall k8s",
			"manual_command": "sudo ai-sre k8s cleanup '<same ofpk8s1 ref>'",
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
	if len(args) >= 2 && args[0] == "k8s" && args[1] == "install" {
		return "可在控制机执行 sudo ai-sre uninstall k8s；如仍持有同一安装引用，也可执行 sudo ai-sre k8s cleanup '<ref>'。"
	}
	if len(args) >= 3 && args[0] == "node" && args[1] == "tune" {
		return "node tune 属于系统配置变更，当前提供人工回滚建议，不自动恢复。"
	}
	return "该命令没有可验证的自动回滚语义。"
}
