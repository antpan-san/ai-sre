package cli

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"time"
)

// maybePromptFeedback asks the user whether the just-printed diagnose answer
// was helpful and forwards the response to OpsFleet so the server-side skill
// pack can refine over time. It is purely additive: any failure path is silent.
//
// Conditions to skip the prompt:
//   - --no-feedback was set
//   - -o json (we don't want to corrupt machine-parseable output)
//   - stdin or stderr is not a TTY (CI / piped invocations)
//   - resolveOpsfleetAPIBase() is empty (no remote target configured)
func maybePromptFeedback(parentCtx context.Context, topic string, diag *diagnoseResponse) {
	if noFeedback {
		return
	}
	if strings.EqualFold(outputFormat, "json") {
		return
	}
	if !isStdinTTY() || !isStderrTTY() {
		return
	}
	base := strings.TrimSpace(resolveOpsfleetAPIBase())
	if base == "" {
		return
	}
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "—— 反馈（可选，回车跳过） ————————————————")
	fmt.Fprintln(os.Stderr, "本次诊断是否帮你定位了根因？输入 y / n / 自由备注；空行跳过。")
	fmt.Fprint(os.Stderr, "> ")
	reader := bufio.NewReader(os.Stdin)
	line, _ := reader.ReadString('\n')
	line = strings.TrimSpace(line)
	if line == "" {
		fmt.Fprintln(os.Stderr, "(已跳过)")
		return
	}
	var helpfulPtr *bool
	note := ""
	switch strings.ToLower(line) {
	case "y", "yes":
		v := true
		helpfulPtr = &v
	case "n", "no":
		v := false
		helpfulPtr = &v
		fmt.Fprint(os.Stderr, "可选：再输一条说明（哪里不准 / 缺什么证据），回车跳过：\n> ")
		line2, _ := reader.ReadString('\n')
		note = strings.TrimSpace(line2)
	default:
		note = line
	}
	skill := ""
	reqID := ""
	if diag != nil {
		skill = strings.TrimSpace(diag.SkillName)
		reqID = strings.TrimSpace(diag.RequestID())
	}
	ctx, cancel := context.WithTimeout(parentCtx, 10*time.Second)
	defer cancel()
	if err := callServerSkillsFeedback(ctx, topic, skill, reqID, helpfulPtr, note); err != nil {
		fmt.Fprintf(os.Stderr, "(反馈上报失败: %v)\n", err)
		return
	}
	fmt.Fprintln(os.Stderr, "已记录到服务端，可执行 ai-sre skills refine --topic "+topic+" 让 LLM 据此精炼技能包。")
}

func isStdinTTY() bool {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}
