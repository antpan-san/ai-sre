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
)

type cliFeedbackAnalyzeResponse struct {
	FeedbackID           string `json:"feedback_id"`
	Classification       string `json:"classification"`
	NeedIteration        bool   `json:"need_iteration"`
	UserMessage          string `json:"user_message"`
	NextAction           string `json:"next_action"`
	Action               string `json:"action,omitempty"`
	AutoIterationCreated bool   `json:"auto_iteration_created"`
	AutoIterationID      string `json:"auto_iteration_id,omitempty"`
}

// callCLIFeedbackAnalyze submits diagnostic feedback for platform auto-iteration triage.
// Uses CLI binding token only (not user JWT). Failures are returned to the caller.
func callCLIFeedbackAnalyze(ctx context.Context, topic, command, summary string, extra map[string]interface{}) (*cliFeedbackAnalyzeResponse, error) {
	base := strings.TrimSpace(resolveOpsfleetAPIBase())
	if base == "" {
		return nil, errors.New("opsfleet api base is empty")
	}
	if strings.TrimSpace(resolveOpsfleetToken()) == "" || strings.TrimSpace(resolveOpsfleetFingerprint()) == "" {
		return nil, errors.New("opsfleet cli binding not configured")
	}
	root := newRoot(progName)
	body, err := json.Marshal(map[string]interface{}{
		"topic":                  strings.TrimSpace(topic),
		"command":                strings.TrimSpace(command),
		"summary":                strings.TrimSpace(summary),
		"context":                extra,
		"command_catalog_digest": CommandCatalogDigest(root),
	})
	if err != nil {
		return nil, err
	}
	endpoint := strings.TrimRight(base, "/") + "/api/cli/feedback/analyze"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	attachOpsfleetAuth(req)
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("cli feedback analyze status=%d: %s", resp.StatusCode, parseOpsfleetErrMsg(raw))
	}
	var env struct {
		Code int                            `json:"code"`
		Msg  string                         `json:"msg"`
		Data cliFeedbackAnalyzeResponse     `json:"data"`
	}
	if err := json.Unmarshal(raw, &env); err != nil {
		return nil, err
	}
	if env.Code != 200 {
		return nil, fmt.Errorf("api code=%d msg=%s", env.Code, strings.TrimSpace(env.Msg))
	}
	return &env.Data, nil
}
