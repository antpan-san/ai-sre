package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

type executorAgentOptions struct {
	APIURL      string
	ClientID    string
	Interval    time.Duration
	LogDir      string
	Once        bool
	AllowNoRoot bool
	MaxOutputKB int
}

type executorHeartbeatRequest struct {
	ClientID      string           `json:"client_id"`
	Fingerprint   string           `json:"fingerprint"`
	HeartbeatTime int64            `json:"heartbeat_time"`
	ClientVersion string           `json:"client_version"`
	ProcessID     int              `json:"process_id"`
	Status        string           `json:"status"`
	LocalIP       string           `json:"local_ip"`
	OSInfo        string           `json:"os_info"`
	PrimaryHost   executorHostInfo `json:"primary_host"`
}

type executorHostInfo struct {
	IP       string `json:"ip"`
	Hostname string `json:"hostname"`
	OSInfo   string `json:"os_info"`
	Status   string `json:"status"`
}

type executorHeartbeatResponse struct {
	Message  string                 `json:"message"`
	Commands []executorAgentCommand `json:"commands"`
}

type executorAgentCommand struct {
	TaskID    string          `json:"task_id"`
	SubTaskID string          `json:"sub_task_id"`
	Command   string          `json:"command"`
	Payload   json.RawMessage `json:"payload"`
	Timeout   int             `json:"timeout"`
}

type executorCommandResult struct {
	TaskID    string `json:"task_id"`
	SubTaskID string `json:"sub_task_id"`
	ClientID  string `json:"client_id"`
	Status    string `json:"status"`
	Output    string `json:"output"`
	ExitCode  int    `json:"exit_code"`
	Error     string `json:"error,omitempty"`
}

func executorAgentCmd() *cobra.Command {
	var opts executorAgentOptions
	cmd := &cobra.Command{
		Use:   "agent",
		Short: "常驻执行器：通过 OpsFleet 心跳协议拉取并执行任务",
		Long: `作为受管机上的轻量执行面运行，复用 OpsFleet 现有心跳协议。

支持首批命令：run_shell、sys_init、time_sync、security_harden、disk_optimize、install_monitor、sync_nodes。`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runExecutorAgent(cmd.Context(), opts)
		},
	}
	cmd.Flags().StringVar(&opts.APIURL, "api-url", "", "OpsFleet API base，例如 http://host:9080/ft-api；默认读取 OPSFLEET_API_URL/安装记录")
	cmd.Flags().StringVar(&opts.ClientID, "client-id", "", "执行器 client_id；默认 hostname + local ip")
	cmd.Flags().DurationVar(&opts.Interval, "interval", 5*time.Second, "心跳间隔")
	cmd.Flags().StringVar(&opts.LogDir, "log-dir", "/var/log/opsfleet-executor", "本地任务日志目录")
	cmd.Flags().BoolVar(&opts.Once, "once", false, "只心跳并处理一轮任务后退出，便于测试")
	cmd.Flags().BoolVar(&opts.AllowNoRoot, "allow-no-root", false, "允许非 root 运行；需要 root 的任务会自行失败")
	cmd.Flags().IntVar(&opts.MaxOutputKB, "max-output-kb", 256, "单任务回传输出最大 KB，完整输出仍写本地日志")
	return cmd
}

func runExecutorAgent(ctx context.Context, opts executorAgentOptions) error {
	base := strings.TrimRight(strings.TrimSpace(opts.APIURL), "/")
	if base == "" {
		base = strings.TrimRight(strings.TrimSpace(resolveOpsfleetAPIBase()), "/")
	}
	if base == "" {
		return errors.New("未配置 OpsFleet API base")
	}
	if opts.Interval <= 0 {
		opts.Interval = 5 * time.Second
	}
	if opts.MaxOutputKB <= 0 {
		opts.MaxOutputKB = 256
	}
	clientID := strings.TrimSpace(opts.ClientID)
	if clientID == "" {
		clientID = defaultExecutorClientID()
	}
	if clientID == "" {
		return errors.New("无法生成 client_id，请显式传入 --client-id")
	}
	if os.Geteuid() != 0 && !opts.AllowNoRoot {
		return errors.New("opsfleet-executor agent 默认需要 root 运行；测试可加 --allow-no-root")
	}
	_ = os.MkdirAll(opts.LogDir, 0755)
	client := &http.Client{Timeout: 30 * time.Second}
	for {
		commands, err := agentHeartbeat(ctx, client, base, clientID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[agent] heartbeat failed: %v\n", err)
		} else {
			for _, command := range commands {
				executeAgentCommand(ctx, client, base, clientID, opts, command)
			}
		}
		if opts.Once {
			return err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(opts.Interval):
		}
	}
}

func agentHeartbeat(ctx context.Context, client *http.Client, base, clientID string) ([]executorAgentCommand, error) {
	host, _ := os.Hostname()
	ip := firstNonLoopbackIPv4()
	reqBody := executorHeartbeatRequest{
		ClientID:      clientID,
		Fingerprint:   "executor:" + computeOpsfleetFingerprint(),
		HeartbeatTime: time.Now().UnixMilli(),
		ClientVersion: Version,
		ProcessID:     os.Getpid(),
		Status:        "normal",
		LocalIP:       ip,
		OSInfo:        runtime.GOOS + " " + runtime.GOARCH,
		PrimaryHost:   executorHostInfo{IP: ip, Hostname: host, OSInfo: runtime.GOOS + " " + runtime.GOARCH, Status: "up"},
	}
	raw, _ := json.Marshal(reqBody)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/api/v1/heartbeats", bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if resp.StatusCode/100 != 2 {
		return nil, fmt.Errorf("heartbeat HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var out executorHeartbeatResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, err
	}
	return out.Commands, nil
}

func executeAgentCommand(ctx context.Context, client *http.Client, base, clientID string, opts executorAgentOptions, cmd executorAgentCommand) {
	_ = postAgentTaskLog(ctx, client, base, cmd, clientID, "info", "executor picked command "+cmd.Command)
	res := runAgentCommand(ctx, opts, cmd)
	res.ClientID = clientID
	if err := postAgentResult(ctx, client, base, res); err != nil {
		fmt.Fprintf(os.Stderr, "[agent] report result failed task=%s sub=%s: %v\n", cmd.TaskID, cmd.SubTaskID, err)
	}
}

func runAgentCommand(parent context.Context, opts executorAgentOptions, cmd executorAgentCommand) executorCommandResult {
	res := executorCommandResult{TaskID: cmd.TaskID, SubTaskID: cmd.SubTaskID, Status: "failed", ExitCode: 1}
	script, err := scriptForAgentCommand(cmd)
	if err != nil {
		res.Error = err.Error()
		return res
	}
	timeout := time.Duration(cmd.Timeout) * time.Second
	if timeout <= 0 {
		timeout = 300 * time.Second
	}
	ctx, cancel := context.WithTimeout(parent, timeout)
	defer cancel()
	execCmd := exec.CommandContext(ctx, "bash", "-lc", script)
	var out bytes.Buffer
	execCmd.Stdout = &out
	execCmd.Stderr = &out
	err = execCmd.Run()
	full := out.String()
	logPath := writeAgentCommandLog(opts.LogDir, cmd, full)
	if ctx.Err() == context.DeadlineExceeded {
		res.Status = "failed"
		res.ExitCode = 124
		res.Error = "timeout"
	} else if err != nil {
		res.Status = "failed"
		res.Error = err.Error()
		if exitErr, ok := err.(*exec.ExitError); ok {
			res.ExitCode = exitErr.ExitCode()
		}
	} else {
		res.Status = "success"
		res.ExitCode = 0
	}
	res.Output = truncateAgentOutput(full, opts.MaxOutputKB)
	if logPath != "" {
		res.Output = strings.TrimRight(res.Output, "\n") + "\n[local_log] " + logPath
	}
	return res
}

func scriptForAgentCommand(cmd executorAgentCommand) (string, error) {
	var payload map[string]interface{}
	if len(cmd.Payload) > 0 && string(cmd.Payload) != "null" {
		if err := json.Unmarshal(cmd.Payload, &payload); err != nil {
			return "", fmt.Errorf("invalid payload: %w", err)
		}
	}
	script, _ := payload["script"].(string)
	switch strings.TrimSpace(cmd.Command) {
	case "run_shell", "sys_init", "time_sync", "security_harden", "disk_optimize", "install_monitor":
		if strings.TrimSpace(script) == "" {
			return "", errors.New("payload.script required")
		}
		return script, nil
	case "sync_nodes":
		return "printf '%s\\n' '{\"workers\":[]}'", nil
	default:
		return "", fmt.Errorf("unsupported command %q", cmd.Command)
	}
}

func postAgentTaskLog(ctx context.Context, client *http.Client, base string, cmd executorAgentCommand, clientID, level, message string) error {
	payload := map[string]string{"task_id": cmd.TaskID, "sub_task_id": cmd.SubTaskID, "client_id": clientID, "level": level, "message": message}
	return postAgentJSON(ctx, client, base+"/api/v1/task/log", payload)
}

func postAgentResult(ctx context.Context, client *http.Client, base string, res executorCommandResult) error {
	return postAgentJSON(ctx, client, base+"/api/v1/task/report", res)
}

func postAgentJSON(ctx context.Context, client *http.Client, endpoint string, payload interface{}) error {
	raw, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if resp.StatusCode/100 != 2 {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return nil
}

func writeAgentCommandLog(dir string, cmd executorAgentCommand, content string) string {
	if strings.TrimSpace(dir) == "" {
		return ""
	}
	_ = os.MkdirAll(dir, 0755)
	name := time.Now().Format("20060102-150405") + "-" + sanitizeAgentLogPart(cmd.SubTaskID) + ".log"
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		return ""
	}
	return path
}

func sanitizeAgentLogPart(s string) string {
	if strings.TrimSpace(s) == "" {
		return "task"
	}
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			b.WriteRune(r)
		}
	}
	if b.Len() == 0 {
		return "task"
	}
	return b.String()
}

func truncateAgentOutput(s string, maxKB int) string {
	limit := maxKB * 1024
	if limit <= 0 || len(s) <= limit {
		return s
	}
	return s[:limit] + "\n[truncated] output exceeded limit"
}

func defaultExecutorClientID() string {
	host, _ := os.Hostname()
	ip := firstNonLoopbackIPv4()
	parts := []string{"executor"}
	if strings.TrimSpace(host) != "" {
		parts = append(parts, strings.TrimSpace(host))
	}
	if strings.TrimSpace(ip) != "" {
		parts = append(parts, strings.TrimSpace(ip))
	}
	return strings.Join(parts, ":")
}

func firstNonLoopbackIPv4() string {
	ifaces, err := net.Interfaces()
	if err != nil {
		return ""
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip4 := ip.To4(); ip4 != nil && !ip4.IsLoopback() {
				return ip4.String()
			}
		}
	}
	return ""
}
