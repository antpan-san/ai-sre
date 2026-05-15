package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

type opsfleetStdEnvelope struct {
	Code int             `json:"code"`
	Data json.RawMessage `json:"data"`
	Msg  string          `json:"msg"`
}

func decodeOpsfleetResponse(body []byte, into interface{}) error {
	var env opsfleetStdEnvelope
	if err := json.Unmarshal(body, &env); err != nil {
		return json.Unmarshal(body, into)
	}
	if env.Code != 200 {
		return fmt.Errorf("api code=%d msg=%s", env.Code, strings.TrimSpace(env.Msg))
	}
	if len(env.Data) == 0 || string(env.Data) == "null" {
		return errors.New("api 响应 data 为空")
	}
	return json.Unmarshal(env.Data, into)
}

func jobCmd() *cobra.Command {
	var (
		machinesCSV string
		timeoutSec  int
		commandFlag string
		wait        bool
		maxWait     time.Duration
		printURL    bool
	)
	run := &cobra.Command{
		Use:   "run",
		Short: "调用 OpsFleet 作业中心，对多台在线机器批量执行 shell",
		Long: fmt.Sprintf(`需要已配置 OpsFleet API 与令牌（与 k8s download 相同）：
  · 环境变量 OPSFLEET_API_URL（须含 /ft-api，如 %[1]s）
  · OPSFLEET_TOKEN 或安装脚本写入的 ~/.config/ai-sre/opsfleet_token

执行后会在终端打印各机输出；使用 --print-console-url 输出带 jobId 的控制台链接，
浏览器打开作业中心即可查看与同页一致的汇总。

machine_ids 须为控制台「在线」机器的 UUID。`, EmbeddedOpsfleetAPIBase),
		RunE: func(cmd *cobra.Command, args []string) error {
			base := strings.TrimRight(strings.TrimSpace(resolveOpsfleetAPIBase()), "/")
			tok := resolveOpsfleetToken()
			if base == "" {
				return errors.New("未配置 OPSFLEET_API_URL 或 ~/.config/ai-sre/opsfleet_api_url")
			}
			if tok == "" {
				return errors.New("未配置 OPSFLEET_TOKEN / opsfleet_token，无法调用 /api/job/execute")
			}
			command := strings.TrimSpace(commandFlag)
			if command == "" || command == "-" {
				b, err := io.ReadAll(os.Stdin)
				if err != nil {
					return fmt.Errorf("读取标准输入: %w", err)
				}
				command = strings.TrimSpace(string(b))
			}
			if command == "" {
				return errors.New("请使用 -c '命令或脚本' 或管道传入脚本内容")
			}
			ids := parseCSVJobMachines(machinesCSV)
			if len(ids) == 0 {
				return errors.New("--machines 不能为空（逗号分隔的机器 UUID）")
			}
			if timeoutSec < 10 {
				timeoutSec = 10
			}
			if timeoutSec > 3600 {
				timeoutSec = 3600
			}

			ctx := cmd.Context()
			jobID, err := opsfleetJobExecute(ctx, base, tok, ids, command, timeoutSec)
			if err != nil {
				return err
			}
			if printURL {
				if u := jobCenterConsoleURL(base, jobID); u != "" {
					_, _ = fmt.Fprintf(cmd.OutOrStdout(), "控制台查看结果（需登录）: %s\n", u)
				}
			}
			if !wait {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "jobId=%s（未等待结果：默认 --wait；或浏览器打开控制台链接）\n", jobID)
				return nil
			}

			results, taskStatus, err := opsfleetJobPollResults(ctx, base, tok, jobID, maxWait)
			if err != nil {
				return err
			}
			printJobResults(cmd.OutOrStdout(), results, taskStatus)
			allOK := true
			for _, r := range results {
				if strings.TrimSpace(r.Status) != "success" {
					allOK = false
					break
				}
			}
			if !allOK {
				return errors.New("部分子任务未成功（见上方输出）")
			}
			return nil
		},
	}
	run.Flags().StringVar(&machinesCSV, "machines", "", "逗号分隔的机器 UUID（必填）")
	run.Flags().IntVar(&timeoutSec, "timeout", 120, "单任务超时秒数（10–3600）")
	run.Flags().StringVarP(&commandFlag, "command", "c", "", "shell 脚本/命令；- 表示从 stdin 读入")
	run.Flags().BoolVar(&wait, "wait", true, "等待各 Agent 回传后再退出")
	run.Flags().DurationVar(&maxWait, "max-wait", 15*time.Minute, "轮询结果的最长等待时间")
	run.Flags().BoolVar(&printURL, "print-console-url", false, "打印浏览器打开的作业中心 URL（含 jobId）")

	cmd := &cobra.Command{
		Use:   "job",
		Short: "OpsFleet 作业中心（批量远程 shell）",
	}
	cmd.AddCommand(run)
	cmd.Example = fmt.Sprintf(`  %[1]s job run --machines "$(uuid1),$(uuid2)" -c 'hostname && uptime'
  %[1]s job run --machines "$(uuid)" --timeout 300 --print-console-url -c 'sudo systemctl reload nginx'
  cat fix.sh | %[1]s job run --machines "$(uuid)" -c -`, progName)

	return cmd
}

func parseCSVJobMachines(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	var out []string
	for _, p := range strings.Split(s, ",") {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func opsfleetHTTP(ctx context.Context, method, apiBase, path string, tok string, body []byte) (*http.Response, error) {
	u, err := url.JoinPath(apiBase, path)
	if err != nil {
		return nil, err
	}
	var rdr io.Reader
	if len(body) > 0 {
		rdr = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, u, rdr)
	if err != nil {
		return nil, err
	}
	if len(body) > 0 {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(tok))
	client := &http.Client{Timeout: 120 * time.Second}
	return client.Do(req)
}

func opsfleetJobExecute(ctx context.Context, apiBase, tok string, machineIDs []string, command string, timeout int) (jobID string, err error) {
	payload := map[string]interface{}{
		"machine_ids": machineIDs,
		"command":     command,
		"timeout":     timeout,
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	resp, err := opsfleetHTTP(ctx, http.MethodPost, apiBase, "/api/job/execute", tok, raw)
	if err != nil {
		return "", fmt.Errorf("POST /api/job/execute: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("job execute HTTP %d: %s", resp.StatusCode, truncateForCliErr(body))
	}
	var data struct {
		JobID  string `json:"jobId"`
		Status string `json:"status"`
	}
	if err := decodeOpsfleetResponse(body, &data); err != nil {
		return "", err
	}
	if strings.TrimSpace(data.JobID) == "" {
		return "", errors.New("响应中缺少 jobId")
	}
	return data.JobID, nil
}

type jobResultRow struct {
	MachineID   string `json:"machine_id"`
	MachineName string `json:"machine_name"`
	MachineIP   string `json:"machine_ip"`
	Status      string `json:"status"`
	Output      string `json:"output"`
	ExitCode    *int   `json:"exit_code"`
	Error       string `json:"error"`
}

func opsfleetJobPollResults(ctx context.Context, apiBase, tok, jobID string, maxWait time.Duration) ([]jobResultRow, string, error) {
	deadline := time.Now().Add(maxWait)
	var last []jobResultRow
	var lastStatus string
	terminal := func(s string) bool {
		switch strings.TrimSpace(s) {
		case "success", "failed", "cancelled", "timeout":
			return true
		default:
			return false
		}
	}
	pollClient := &http.Client{Timeout: 30 * time.Second}
	for time.Now().Before(deadline) {
		u, err := url.JoinPath(apiBase, "/api/job/result/"+jobID)
		if err != nil {
			return nil, "", err
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
		if err != nil {
			return nil, "", err
		}
		req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(tok))
		resp, err := pollClient.Do(req)
		if err != nil {
			return nil, "", err
		}
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
		resp.Body.Close()
		if resp.StatusCode >= 400 {
			return nil, "", fmt.Errorf("GET job result HTTP %d: %s", resp.StatusCode, truncateForCliErr(body))
		}
		var envelope struct {
			JobID   string         `json:"jobId"`
			Status  string         `json:"status"`
			Results []jobResultRow `json:"results"`
		}
		if err := decodeOpsfleetResponse(body, &envelope); err != nil {
			return nil, "", err
		}
		last = envelope.Results
		lastStatus = envelope.Status
		if len(last) == 0 {
			time.Sleep(800 * time.Millisecond)
			continue
		}
		allDone := true
		for _, r := range last {
			if !terminal(r.Status) {
				allDone = false
				break
			}
		}
		if allDone {
			return last, lastStatus, nil
		}
		time.Sleep(900 * time.Millisecond)
	}
	return last, lastStatus, fmt.Errorf("等待结果超时（任务可能仍在执行；jobId=%s）", jobID)
}

func truncateForCliErr(b []byte) string {
	s := strings.TrimSpace(string(b))
	if len(s) > 480 {
		return s[:480] + "…"
	}
	return s
}

func printJobResults(w io.Writer, rows []jobResultRow, taskStatus string) {
	_, _ = fmt.Fprintf(w, "task_status=%s\n", taskStatus)
	for i, r := range rows {
		_, _ = fmt.Fprintf(w, "\n━━━ #%d %s (%s) id=%s status=%s exit=%v ━━━\n",
			i+1, r.MachineName, r.MachineIP, r.MachineID, r.Status, derefExitJob(r.ExitCode))
		if strings.TrimSpace(r.Error) != "" {
			_, _ = fmt.Fprintf(w, "stderr/错误: %s\n", strings.TrimSpace(r.Error))
		}
		out := strings.TrimRight(r.Output, "\n")
		if out != "" {
			_, _ = fmt.Fprintln(w, out)
		}
	}
}

func derefExitJob(p *int) string {
	if p == nil {
		return "-"
	}
	return fmt.Sprintf("%d", *p)
}

func jobCenterConsoleOrigin(apiBase string) string {
	b := strings.TrimRight(strings.TrimSpace(apiBase), "/")
	suf := "/ft-api"
	if len(b) > len(suf) && strings.EqualFold(b[len(b)-len(suf):], suf) {
		return b[:len(b)-len(suf)]
	}
	if v := strings.TrimSpace(os.Getenv("OPSFLEET_UI_ORIGIN")); v != "" {
		return strings.TrimRight(v, "/")
	}
	return ""
}

func jobCenterConsoleURL(apiBase, jobID string) string {
	org := jobCenterConsoleOrigin(apiBase)
	if org == "" || strings.TrimSpace(jobID) == "" {
		return ""
	}
	u, err := url.Parse(org)
	if err != nil {
		return ""
	}
	u.Path = "/admin/job/center"
	q := u.Query()
	q.Set("jobId", jobID)
	u.RawQuery = q.Encode()
	return u.String()
}
