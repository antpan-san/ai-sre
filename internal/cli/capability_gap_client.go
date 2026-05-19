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

type capabilityGapRequest struct {
	Topic   string                 `json:"topic"`
	Context map[string]string      `json:"context,omitempty"`
	Intent  executionIntent        `json:"intent"`
}

type capabilityGapResponse struct {
	PackKey         string `json:"pack_key"`
	NodePath        string `json:"node_path"`
	Granted         bool   `json:"granted"`
	TreeSynced      bool   `json:"tree_synced"`
	AutoIterationID string `json:"auto_iteration_id,omitempty"`
	Message         string `json:"message,omitempty"`
}

func requestCapabilityGap(ctx context.Context, intent executionIntent, contextKV map[string]string) error {
	base := strings.TrimSpace(resolveOpsfleetAPIBase())
	if base == "" {
		return fmt.Errorf("opsfleet api base is empty")
	}
	if strings.TrimSpace(resolveOpsfleetToken()) == "" {
		return fmt.Errorf("需要绑定 OpsFleet CLI token")
	}
	body, err := json.Marshal(capabilityGapRequest{
		Topic:   intent.Topic,
		Context: contextKV,
		Intent:  intent,
	})
	if err != nil {
		return err
	}
	endpoint := strings.TrimRight(base, "/") + "/api/cli/capability-gap"
	hreq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	hreq.Header.Set("Content-Type", "application/json")
	attachOpsfleetAuth(hreq)
	hreq.Header.Set("X-AI-SRE-Version", strings.TrimSpace(Version))
	client := &http.Client{Timeout: 45 * time.Second}
	resp, err := client.Do(hreq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return err
	}
	if resp.StatusCode >= 300 {
		return fmt.Errorf("capability-gap status=%d: %s", resp.StatusCode, parseOpsfleetErrMsg(raw))
	}
	var env struct {
		Code int             `json:"code"`
		Msg  string          `json:"msg"`
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(raw, &env); err != nil {
		return err
	}
	if env.Code != 200 {
		return fmt.Errorf("api code=%d msg=%s", env.Code, env.Msg)
	}
	var out capabilityGapResponse
	if err := json.Unmarshal(env.Data, &out); err != nil {
		return err
	}
	if !out.Granted {
		return fmt.Errorf("服务端未授予技能包")
	}
	return nil
}

func isCapabilityNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "未在技能树中找到可执行能力")
}
