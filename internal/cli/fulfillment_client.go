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

	"github.com/spf13/cobra"
)

const (
	fulfillmentActionSubscriptionRequired = "subscription_required"
	fulfillmentActionGrantedRetry         = "granted_retry"
	fulfillmentActionManualReview         = "manual_review"
	fulfillmentActionAutoIterationCreated = "auto_iteration_created"
	fulfillmentActionAwaitingApproval     = "awaiting_approval"
	fulfillmentActionUnsupported          = "unsupported"
)

type fulfillmentPlanRequest struct {
	Command              string            `json:"command"`
	CommandCatalogDigest string            `json:"command_catalog_digest"`
	Topic                string            `json:"topic,omitempty"`
	Context              map[string]string `json:"context,omitempty"`
	Intent               executionIntent   `json:"intent"`
	FailureKind          string            `json:"failure_kind"`
	FailureMessage       string            `json:"failure_message,omitempty"`
}

type fulfillmentPlanResponse struct {
	Action               string `json:"action"`
	Message              string `json:"message,omitempty"`
	AutoIterationCreated bool   `json:"auto_iteration_created"`
	AutoIterationID      string `json:"auto_iteration_id,omitempty"`
	RetryAllowed         bool   `json:"retry_allowed"`
}

func requestFulfillmentPlan(ctx context.Context, root *cobra.Command, intent executionIntent, failureKind, failureMessage string, contextKV map[string]string) (*fulfillmentPlanResponse, error) {
	base := strings.TrimSpace(resolveOpsfleetAPIBase())
	if base == "" {
		return nil, fmt.Errorf("opsfleet api base is empty")
	}
	if strings.TrimSpace(resolveOpsfleetToken()) == "" {
		return nil, fmt.Errorf("需要绑定 OpsFleet CLI token")
	}
	digest := CommandCatalogDigest(root)
	cmdPath := strings.TrimSpace(intent.CommandKind)
	if cmdPath == "" {
		cmdPath = "check " + strings.TrimSpace(intent.Topic)
	}
	body, err := json.Marshal(fulfillmentPlanRequest{
		Command:              cmdPath,
		CommandCatalogDigest: digest,
		Topic:                intent.Topic,
		Context:              contextKV,
		Intent:               intent,
		FailureKind:          failureKind,
		FailureMessage:       failureMessage,
	})
	if err != nil {
		return nil, err
	}
	endpoint := strings.TrimRight(base, "/") + "/api/cli/fulfillment/plan"
	hreq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	hreq.Header.Set("Content-Type", "application/json")
	attachOpsfleetAuth(hreq)
	hreq.Header.Set("X-AI-SRE-Version", strings.TrimSpace(Version))
	client := &http.Client{Timeout: 45 * time.Second}
	resp, err := client.Do(hreq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("fulfillment plan status=%d: %s", resp.StatusCode, parseOpsfleetErrMsg(raw))
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
		return nil, fmt.Errorf("api code=%d msg=%s", env.Code, env.Msg)
	}
	var out fulfillmentPlanResponse
	if err := json.Unmarshal(env.Data, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func fulfillmentPlanUserMessage(plan *fulfillmentPlanResponse, fallback error) error {
	if plan == nil {
		if fallback != nil {
			return fallback
		}
		return fmt.Errorf("fulfillment plan 无响应")
	}
	msg := strings.TrimSpace(plan.Message)
	if msg == "" {
		switch plan.Action {
		case fulfillmentActionSubscriptionRequired:
			msg = "当前订阅不足，请联系管理员或购买对应能力"
		case fulfillmentActionGrantedRetry:
			msg = "已授权，请重试命令"
		case fulfillmentActionManualReview:
			msg = "已记录反馈，将由管理员人工处理"
		case fulfillmentActionAutoIterationCreated, fulfillmentActionAwaitingApproval:
			msg = "已创建自动迭代任务"
		case fulfillmentActionUnsupported:
			msg = "当前产品暂不支持该需求，已记录反馈"
		default:
			if fallback != nil {
				return fallback
			}
			msg = "能力无法满足当前请求"
		}
	}
	if plan.AutoIterationID != "" {
		msg += "（任务 " + plan.AutoIterationID + "）"
	}
	return fmt.Errorf("%s", msg)
}
