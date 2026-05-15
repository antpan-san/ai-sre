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
	"strings"
	"time"

	goruntime "github.com/panshuai/ai-sre/internal/go_runtime"
)

const maxRuntimeWatchUploadBytes = 512 * 1024

// postRuntimeWatchSample uploads one watch report to the OpsFleet public ingest endpoint.
func postRuntimeWatchSample(ctx context.Context, endpointURL, sessionID, token string, watch any) error {
	endpointURL = strings.TrimSpace(endpointURL)
	sessionID = strings.TrimSpace(sessionID)
	token = strings.TrimSpace(token)
	if endpointURL == "" || sessionID == "" || token == "" {
		return fmt.Errorf("upload-url、session-id 与 sample-token 均不能为空")
	}
	body := map[string]any{
		"session_id": sessionID,
		"token":      token,
		"watch":      watch,
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return err
	}
	if len(raw) > maxRuntimeWatchUploadBytes {
		return fmt.Errorf("上报 JSON 超过 %d 字节，请减少 --watch-samples 或缩短观测窗口", maxRuntimeWatchUploadBytes)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpointURL, bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("上报失败: HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}
	return nil
}

type goRuntimeReportUploadResult struct {
	ExecutionRecordID     string `json:"execution_record_id"`
	RuntimeWatchSessionID string `json:"runtime_watch_session_id"`
	RuntimeWatchSampleID  string `json:"runtime_watch_sample_id"`
}

func checkGoRuntimeAuth(ctx context.Context, apiBase, token, fingerprint string) error {
	if strings.TrimSpace(apiBase) == "" || strings.TrimSpace(token) == "" || strings.TrimSpace(fingerprint) == "" {
		return fmt.Errorf("缺少 OpsFleet API、CLI token 或机器指纹")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(apiBase, "/")+"/api/cli/go-runtime/auth-check", nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(token))
	req.Header.Set("X-OpsFleet-CLI-Fingerprint", strings.TrimSpace(fingerprint))
	req.Header.Set("X-OpsFleet-CLI-Version", Version)
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}
	var envelope struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
	}
	if err := json.Unmarshal(b, &envelope); err != nil {
		return err
	}
	if envelope.Code != 0 && envelope.Code != 200 {
		if strings.TrimSpace(envelope.Msg) == "" {
			envelope.Msg = "Go runtime 鉴权失败"
		}
		return errors.New(envelope.Msg)
	}
	return nil
}

func postGoRuntimeReport(ctx context.Context, apiBase, token, fingerprint, command string, watch *goruntime.WatchReport) (goRuntimeReportUploadResult, error) {
	var out goRuntimeReportUploadResult
	if strings.TrimSpace(apiBase) == "" || strings.TrimSpace(token) == "" || strings.TrimSpace(fingerprint) == "" {
		return out, fmt.Errorf("缺少 OpsFleet API、CLI token 或机器指纹")
	}
	host, _ := os.Hostname()
	body := map[string]any{
		"command": command,
		"host":    host,
		"watch":   watch,
		"client": map[string]any{
			"version":          Version,
			"binding_id":       resolveOpsfleetBindingID(),
			"fingerprint_hash": fingerprint,
		},
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return out, err
	}
	if len(raw) > maxRuntimeWatchUploadBytes {
		return out, fmt.Errorf("Go runtime 报告超过 %d 字节", maxRuntimeWatchUploadBytes)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(apiBase, "/")+"/api/cli/go-runtime/reports", bytes.NewReader(raw))
	if err != nil {
		return out, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(token))
	req.Header.Set("X-OpsFleet-CLI-Fingerprint", strings.TrimSpace(fingerprint))
	req.Header.Set("X-OpsFleet-CLI-Version", Version)
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return out, err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 128<<10))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return out, fmt.Errorf("HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}
	var envelope struct {
		Code int                         `json:"code"`
		Msg  string                      `json:"msg"`
		Data goRuntimeReportUploadResult `json:"data"`
	}
	if err := json.Unmarshal(b, &envelope); err != nil {
		return out, err
	}
	if envelope.Code != 0 && envelope.Code != 200 {
		return out, errors.New(envelope.Msg)
	}
	return envelope.Data, nil
}
