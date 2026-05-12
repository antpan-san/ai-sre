package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/panshuai/ai-sre/internal/engine"
)

// serverAIResult builds a minimal engine result for ask/runbook/analyze when served by OpsFleet.
func serverAIResult(answer string) *engine.RunResult {
	a := strings.TrimSpace(answer)
	return &engine.RunResult{
		Answer:       a,
		SkillName:    "server-ai",
		SkillDisplay: "OpsFleet 服务端 AI",
	}
}

func decodeDiagnoseResponseFromBody(raw []byte) (*diagnoseResponse, error) {
	var env struct {
		Code int             `json:"code"`
		Msg  string          `json:"msg"`
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(raw, &env); err != nil {
		return nil, err
	}
	if env.Code != 200 {
		return nil, fmt.Errorf("api code=%d msg=%s", env.Code, strings.TrimSpace(env.Msg))
	}
	if len(env.Data) > 0 && string(env.Data) != "null" {
		var out diagnoseResponse
		if err := json.Unmarshal(env.Data, &out); err != nil {
			return nil, err
		}
		return &out, nil
	}
	var flat diagnoseResponse
	if err := json.Unmarshal(raw, &flat); err == nil && strings.TrimSpace(flat.Answer) != "" {
		return &flat, nil
	}
	return nil, errors.New("empty diagnose data from server")
}

func decodeServerSimpleAnswer(raw []byte) (string, error) {
	var env struct {
		Code int             `json:"code"`
		Msg  string          `json:"msg"`
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(raw, &env); err != nil {
		return "", err
	}
	if env.Code != 200 {
		return "", fmt.Errorf("api code=%d msg=%s", env.Code, strings.TrimSpace(env.Msg))
	}
	var data struct {
		Answer string `json:"answer"`
	}
	if len(env.Data) == 0 || string(env.Data) == "null" {
		return "", errors.New("empty data from server")
	}
	if err := json.Unmarshal(env.Data, &data); err != nil {
		return "", err
	}
	return strings.TrimSpace(data.Answer), nil
}

func callServerAsk(ctx context.Context, question string, noRag bool) (string, error) {
	base := strings.TrimSpace(resolveOpsfleetAPIBase())
	if base == "" {
		return "", errors.New("opsfleet api base is empty")
	}
	endpoint := strings.TrimRight(base, "/") + "/api/ai/ask"
	body, err := json.Marshal(map[string]interface{}{
		"question": strings.TrimSpace(question),
		"no_rag":   noRag,
	})
	if err != nil {
		return "", err
	}
	hreq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	hreq.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(hreq)
	if err != nil {
		return "", fmt.Errorf("call server ask: %w", err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return "", err
	}
	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("server ask status=%d: %s", resp.StatusCode, parseOpsfleetErrMsg(raw))
	}
	return decodeServerSimpleAnswer(raw)
}

func callServerRunbook(ctx context.Context, scenario string, ctxMap map[string]string) (string, error) {
	base := strings.TrimSpace(resolveOpsfleetAPIBase())
	if base == "" {
		return "", errors.New("opsfleet api base is empty")
	}
	endpoint := strings.TrimRight(base, "/") + "/api/ai/runbook"
	body, err := json.Marshal(map[string]interface{}{
		"scenario": strings.TrimSpace(scenario),
		"context":  ctxMap,
	})
	if err != nil {
		return "", err
	}
	hreq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	hreq.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 90 * time.Second}
	resp, err := client.Do(hreq)
	if err != nil {
		return "", fmt.Errorf("call server runbook: %w", err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return "", err
	}
	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("server runbook status=%d: %s", resp.StatusCode, parseOpsfleetErrMsg(raw))
	}
	return decodeServerSimpleAnswer(raw)
}

func parseOpsfleetErrMsg(raw []byte) string {
	msg := strings.TrimSpace(string(raw))
	var api struct {
		Msg string `json:"msg"`
	}
	if json.Unmarshal(raw, &api) == nil && strings.TrimSpace(api.Msg) != "" {
		msg = strings.TrimSpace(api.Msg)
	}
	if msg == "" {
		msg = "(empty body)"
	}
	return msg
}
