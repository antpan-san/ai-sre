package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
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
