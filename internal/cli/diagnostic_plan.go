package cli

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

const diagnosticPlanUploadMaxBytes = 512 * 1024

type serverDiagnosticPlanStep struct {
	ID             string   `json:"id"`
	Title          string   `json:"title"`
	Argv           []string `json:"argv"`
	TimeoutSeconds int      `json:"timeout_seconds"`
	EvidenceKey    string   `json:"evidence_key"`
}

type serverDiagnosticPlan struct {
	PlanID               string                     `json:"plan_id"`
	PlanToken            string                     `json:"plan_token"`
	Topic                string                     `json:"topic"`
	ExpiresAt            time.Time                  `json:"expires_at"`
	RequiresConfirmation bool                       `json:"requires_confirmation"`
	Steps                []serverDiagnosticPlanStep `json:"steps"`
}

func shouldRequestServerDiagnosticPlan(topic string, kv map[string]string) bool {
	t := strings.ToLower(strings.TrimSpace(topic))
	switch t {
	case "k8s", "kubernetes":
		return !hasKubectlEvidence(kv)
	case "go_runtime", "go-runtime":
		return !hasGoRuntimeDiagnosticEvidence(kv)
	default:
		return false
	}
}

func hasGoRuntimeDiagnosticEvidence(kv map[string]string) bool {
	if kv == nil {
		return false
	}
	for k := range kv {
		if strings.HasPrefix(k, "go_runtime_") || strings.HasPrefix(k, "host_") {
			return true
		}
	}
	return false
}

func maybeRunServerDiagnosticPlan(ctx context.Context, topic string, kv map[string]string, yes bool) (map[string]string, bool, error) {
	if strings.TrimSpace(resolveOpsfleetAPIBase()) == "" || strings.TrimSpace(resolveOpsfleetToken()) == "" || strings.TrimSpace(resolveOpsfleetFingerprint()) == "" {
		return nil, false, nil
	}
	plan, err := requestServerDiagnosticPlan(ctx, topic, kv)
	if err != nil {
		return nil, false, err
	}
	if plan == nil || len(plan.Steps) == 0 {
		return nil, false, nil
	}
	if err := confirmDiagnosticPlan(plan, yes); err != nil {
		return nil, false, err
	}
	obs := executeDiagnosticPlan(ctx, plan)
	if err := postServerDiagnosticPlanObservations(ctx, plan, obs); err != nil {
		return obs, true, err
	}
	return obs, true, nil
}

func requestServerDiagnosticPlan(ctx context.Context, topic string, kv map[string]string) (*serverDiagnosticPlan, error) {
	body, err := json.Marshal(map[string]interface{}{
		"topic":      strings.TrimSpace(topic),
		"context":    kv,
		"command":    strings.Join(os.Args, " "),
		"request_id": "",
		"client":     opsfleetAIClient(),
	})
	if err != nil {
		return nil, err
	}
	endpoint := strings.TrimRight(resolveOpsfleetAPIBase(), "/") + "/api/cli/diagnostics/plan"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	attachOpsfleetAuth(req)
	resp, err := (&http.Client{Timeout: 30 * time.Second}).Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求服务端诊断任务单失败: %w", err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("服务端诊断任务单 status=%d: %s", resp.StatusCode, parseOpsfleetErrMsg(raw))
	}
	var env struct {
		Code int             `json:"code"`
		Msg  string          `json:"msg"`
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(raw, &env); err != nil {
		return nil, err
	}
	if env.Code != 200 {
		return nil, fmt.Errorf("服务端诊断任务单 code=%d: %s", env.Code, strings.TrimSpace(env.Msg))
	}
	var out serverDiagnosticPlan
	if len(env.Data) > 0 && string(env.Data) != "null" {
		if err := json.Unmarshal(env.Data, &out); err != nil {
			return nil, err
		}
	} else if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	if strings.TrimSpace(out.PlanID) == "" || strings.TrimSpace(out.PlanToken) == "" {
		return nil, errors.New("服务端诊断任务单缺少 plan_id 或 token")
	}
	return &out, nil
}

func postServerDiagnosticPlanObservations(ctx context.Context, plan *serverDiagnosticPlan, obs map[string]string) error {
	if plan == nil {
		return nil
	}
	body, err := json.Marshal(map[string]interface{}{
		"plan_id":      plan.PlanID,
		"plan_token":   plan.PlanToken,
		"observations": obs,
		"summary":      diagnosticObservationSummary(obs),
		"client":       opsfleetAIClient(),
	})
	if err != nil {
		return err
	}
	endpoint := strings.TrimRight(resolveOpsfleetAPIBase(), "/") + "/api/cli/diagnostics/observations"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	attachOpsfleetAuth(req)
	resp, err := (&http.Client{Timeout: 30 * time.Second}).Do(req)
	if err != nil {
		return fmt.Errorf("上报诊断证据失败: %w", err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return err
	}
	if resp.StatusCode >= 300 {
		return fmt.Errorf("上报诊断证据 status=%d: %s", resp.StatusCode, parseOpsfleetErrMsg(raw))
	}
	return nil
}

func confirmDiagnosticPlan(plan *serverDiagnosticPlan, yes bool) error {
	if plan == nil || len(plan.Steps) == 0 || !plan.RequiresConfirmation {
		return nil
	}
	fmt.Fprintln(os.Stderr, "服务端生成了只读诊断任务单，将执行以下命令采集证据：")
	for _, st := range plan.Steps {
		fmt.Fprintf(os.Stderr, "  - %s: %s\n", strings.TrimSpace(st.Title), shellJoinForDisplay(st.Argv))
	}
	if yes {
		return nil
	}
	if !isStdinTTY() {
		return errors.New("服务端诊断任务单需要 --yes 确认")
	}
	fmt.Fprint(os.Stderr, "是否执行这些只读命令？输入 y 回车继续: ")
	line, _ := bufio.NewReader(os.Stdin).ReadString('\n')
	switch strings.ToLower(strings.TrimSpace(line)) {
	case "y", "yes":
		return nil
	default:
		return errors.New("已取消服务端只读诊断任务单")
	}
}

func executeDiagnosticPlan(ctx context.Context, plan *serverDiagnosticPlan) map[string]string {
	col := newEvidenceCollector()
	for _, st := range plan.Steps {
		key := strings.TrimSpace(st.EvidenceKey)
		if key == "" {
			key = strings.TrimSpace(st.ID)
		}
		if key == "" {
			continue
		}
		if !allowedCLIDiagnosticPlanCommand(st.Argv) {
			col.put(key+"_blocked", "blocked unsafe diagnostic plan command: "+shellJoinForDisplay(st.Argv))
			continue
		}
		timeout := time.Duration(st.TimeoutSeconds) * time.Second
		if timeout <= 0 || timeout > 60*time.Second {
			timeout = 20 * time.Second
		}
		out := runDiagnosticPlanCommand(ctx, timeout, st.Argv)
		col.put(key, out)
		if len(marshalStringMap(col.out)) >= diagnosticPlanUploadMaxBytes {
			break
		}
	}
	return col.out
}

func runDiagnosticPlanCommand(ctx context.Context, timeout time.Duration, argv []string) string {
	cctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	execArgv := append([]string(nil), argv...)
	if execArgv[0] == "ai-sre" {
		if self := selfExecutablePath(); self != "" {
			execArgv[0] = self
		}
	}
	cmd := exec.CommandContext(cctx, execArgv[0], execArgv[1:]...)
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	err := cmd.Run()
	out := strings.TrimSpace(buf.String())
	if err != nil {
		if out == "" {
			return fmt.Sprintf("[%s failed: %v]", shellJoinForDisplay(argv), err)
		}
		return out + fmt.Sprintf("\n[exit error: %v]", err)
	}
	return out
}

func diagnosticObservationSummary(obs map[string]string) string {
	if len(obs) == 0 {
		return "no observations"
	}
	keys := make([]string, 0, len(obs))
	total := 0
	for k, v := range obs {
		keys = append(keys, k)
		total += len(k) + len(v)
	}
	return fmt.Sprintf("keys=%s bytes=%d", strings.Join(uniqueSortedStrings(keys), ","), total)
}

func allowedCLIDiagnosticPlanCommand(argv []string) bool {
	if len(argv) == 0 {
		return false
	}
	if argv[0] == "ai-sre" || strings.HasSuffix(argv[0], "/ai-sre") {
		return allowedCLIAISreDiagnosticCommand(argv)
	}
	if argv[0] != "kubectl" {
		return false
	}
	for _, a := range argv {
		if strings.TrimSpace(a) == "" || strings.ContainsAny(a, ";&|`$<>") {
			return false
		}
	}
	if len(argv) < 2 {
		return false
	}
	switch argv[1] {
	case "version":
		return diagnosticArgsSubset(argv[2:], []string{"--client=true", "-o", "yaml", "json"})
	case "config":
		return len(argv) == 3 && argv[2] == "current-context"
	case "get":
		return allowedCLIKubectlGet(argv[2:])
	case "describe":
		return allowedCLIKubectlDescribe(argv[2:])
	case "logs":
		return allowedCLIKubectlLogs(argv[2:])
	default:
		return false
	}
}

func allowedCLIKubectlGet(args []string) bool {
	if len(args) == 0 {
		return false
	}
	switch args[0] {
	case "nodes", "pods", "pod", "events":
	default:
		return false
	}
	return diagnosticArgsSubset(args[1:], []string{"-A", "--all-namespaces", "-n", "--namespace", "-o", "wide", "json", "yaml", "--sort-by=.metadata.creationTimestamp"}) &&
		diagnosticK8sFlagValuesAllowed(args[1:])
}

func allowedCLIKubectlDescribe(args []string) bool {
	return len(args) > 0 && args[0] == "pod" &&
		diagnosticArgsSubset(args[1:], []string{"-n", "--namespace"}) &&
		diagnosticK8sFlagValuesAllowed(args[1:])
}

func allowedCLIKubectlLogs(args []string) bool {
	return diagnosticArgsSubset(args, []string{"-n", "--namespace", "--all-containers=true", "--previous"}) &&
		diagnosticK8sFlagValuesAllowed(args)
}

func diagnosticArgsSubset(args []string, allowed []string) bool {
	set := map[string]struct{}{}
	for _, a := range allowed {
		set[a] = struct{}{}
	}
	for _, a := range args {
		if strings.HasPrefix(a, "--field-selector=") || strings.HasPrefix(a, "--tail=") {
			continue
		}
		if strings.HasPrefix(a, "-") {
			if _, ok := set[a]; !ok {
				return false
			}
		}
	}
	return true
}

func diagnosticK8sFlagValuesAllowed(args []string) bool {
	expectValue := ""
	for _, a := range args {
		if expectValue != "" {
			if !diagnosticK8sSafeNameRe.MatchString(a) && a != "wide" && a != "json" && a != "yaml" {
				return false
			}
			expectValue = ""
			continue
		}
		switch a {
		case "-n", "--namespace", "-o":
			expectValue = a
		default:
			if strings.HasPrefix(a, "--field-selector=") {
				v := strings.TrimPrefix(a, "--field-selector=")
				if !strings.HasPrefix(v, "involvedObject.name=") && !strings.HasPrefix(v, "metadata.name=") && !strings.HasPrefix(v, "status.phase=") {
					return false
				}
				if strings.ContainsAny(v, ";&|`$<>") {
					return false
				}
			}
			if strings.HasPrefix(a, "--tail=") && !diagnosticTailRe.MatchString(a) {
				return false
			}
		}
	}
	return expectValue == ""
}

func shellJoinForDisplay(argv []string) string {
	parts := make([]string, 0, len(argv))
	for _, a := range argv {
		if strings.ContainsAny(a, " \t\n\"'") {
			parts = append(parts, strconvQuote(a))
		} else {
			parts = append(parts, a)
		}
	}
	return strings.Join(parts, " ")
}

func strconvQuote(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

func marshalStringMap(v map[string]string) []byte {
	b, _ := json.Marshal(v)
	return b
}

func allowedCLIAISreDiagnosticCommand(argv []string) bool {
	if len(argv) < 3 || argv[1] != "go_runtime" || argv[2] != "diagnose" {
		return false
	}
	for _, a := range argv {
		if strings.TrimSpace(a) == "" || strings.ContainsAny(a, ";&|`$<>") {
			return false
		}
	}
	return allowedCLIAISreArgs(argv[3:])
}

func allowedCLIAISreArgs(args []string) bool {
	allowed := map[string]struct{}{
		"--json": {}, "--pod": {}, "--deployment": {}, "--statefulset": {}, "--daemonset": {},
		"--replicaset": {}, "--job": {}, "--cronjob": {}, "--service": {}, "--ingress": {}, "--pvc": {},
		"--pid": {}, "--name": {}, "--pid-name": {},
	}
	expectValue := ""
	for _, a := range args {
		if expectValue != "" {
			if !diagnosticAISreValueRe.MatchString(a) {
				return false
			}
			expectValue = ""
			continue
		}
		if !strings.HasPrefix(a, "--") {
			return false
		}
		if _, ok := allowed[a]; !ok {
			return false
		}
		if a != "--json" {
			expectValue = a
		}
	}
	return expectValue == ""
}

var (
	diagnosticK8sSafeNameRe = regexp.MustCompile(`^[A-Za-z0-9_.-]{1,253}$`)
	diagnosticTailRe        = regexp.MustCompile(`^--tail=[0-9]{1,5}$`)
	diagnosticAISreValueRe  = regexp.MustCompile(`^[A-Za-z0-9_./:-]{0,512}$`)
)
