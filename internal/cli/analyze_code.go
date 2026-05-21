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

	"github.com/spf13/cobra"
)

// 与 ft-backend handlers.ErrorCodeAnalyze 响应字段对齐
type errorCodeCardResponse struct {
	Code             string   `json:"code"`
	Summary          string   `json:"summary"`
	RootCause        string   `json:"root_cause"`
	TypicalEvidence  []string `json:"typical_evidence"`
	RecoveryOneLiner string   `json:"recovery_one_liner"`
	PlatformFollowup string   `json:"platform_followup"`
	RelatedCodes     []string `json:"related_codes"`
	Source           string   `json:"source"`
	SkillName        string   `json:"skill_name"`
	SkillSource      string   `json:"skill_source"`
	DetailEcho       string   `json:"detail_echo"`
}

func analyzeCodeCmd() *cobra.Command {
	var detail string
	var listAll bool
	cmd := &cobra.Command{
		Use:   "code [CODE]",
		Short: "把部署/运行错误码翻译成根因卡片（无 LLM 调用，纯查目录；不命中时由服务端推断）",
		Long: `仅给根因，不给排查清单。

CODE 形如 OPSFLEET_K8S_E_PAUSE_MISSING、OPSFLEET_DL_E_NETWORK、OPSFLEET_K8S_I_RELAY_ROUTE_APPLIED、OPSFLEET_K8S_E_APISERVER_TIMEOUT。
来源：ft-backend skills/builtin/error_codes.yaml（运维侧可通过 ai-sre expert skills server 列出）。

工作流：
  1. ai-sre 命中本机错误码索引（如适用，未来 0.5.x） → 直接打印；
  2. 否则向控制台 /api/ai/error-codes/analyze 发起一次 GET-like POST，O(<1ms) 查目录；
  3. 完全未命中则回退到服务端 LLM 推断（暂未启用，目前返回 fallback_unknown）。

也可以用 --list 列出所有错误码（用于客户端排错速查表）。`,
		Args: func(c *cobra.Command, args []string) error {
			if listAll {
				return nil
			}
			if len(args) < 1 {
				return errors.New("必须提供 CODE，或使用 --list 列出全部")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			if ctx == nil {
				ctx = context.Background()
			}
			if listAll {
				return runListErrorCodes(ctx)
			}
			code := strings.TrimSpace(strings.ToUpper(args[0]))
			return runAnalyzeErrorCode(ctx, code, detail)
		},
	}
	cmd.Flags().StringVar(&detail, "detail", "", "可选：粘贴一段错误原文，服务端会原样回显并供 LLM fallback 使用")
	cmd.Flags().BoolVar(&listAll, "list", false, "列出全部错误码")
	return cmd
}

func runListErrorCodes(ctx context.Context) error {
	base := strings.TrimSpace(resolveOpsfleetAPIBase())
	if base == "" {
		return errors.New("未配置 OpsFleet API（OPSFLEET_API_URL）也未内置默认值；无法访问错误码目录")
	}
	url := strings.TrimRight(base, "/") + "/api/ai/error-codes"
	body, err := opsfleetHTTPGet(ctx, url)
	if err != nil {
		return err
	}
	var env struct {
		Code int             `json:"code"`
		Msg  string          `json:"msg"`
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(body, &env); err != nil {
		return fmt.Errorf("解析错误码目录响应失败: %w", err)
	}
	if env.Code != 0 && env.Code != 200 {
		return fmt.Errorf("服务端错误: %s", env.Msg)
	}
	var data struct {
		Codes []errorCodeCardResponse `json:"codes"`
		Count int                     `json:"count"`
	}
	if err := json.Unmarshal(env.Data, &data); err != nil {
		return fmt.Errorf("解析 data 失败: %w", err)
	}
	if outputFormat == "json" {
		raw, _ := json.MarshalIndent(data, "", "  ")
		fmt.Println(string(raw))
		return nil
	}
	fmt.Printf("# 错误码目录（%d 条）— 详见 ai-sre analyze code <CODE>\n\n", data.Count)
	for _, ec := range data.Codes {
		fmt.Printf("- %s\n  %s\n", ec.Code, strings.TrimSpace(ec.Summary))
	}
	return nil
}

func runAnalyzeErrorCode(ctx context.Context, code, detail string) error {
	base := strings.TrimSpace(resolveOpsfleetAPIBase())
	if base == "" {
		return errors.New("未配置 OpsFleet API（OPSFLEET_API_URL）也未内置默认值；无法访问错误码目录")
	}
	url := strings.TrimRight(base, "/") + "/api/ai/error-codes/analyze"
	payload, _ := json.Marshal(map[string]string{"code": code, "detail": detail})
	body, err := opsfleetHTTPPost(ctx, url, payload)
	if err != nil {
		return err
	}
	var env struct {
		Code int             `json:"code"`
		Msg  string          `json:"msg"`
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(body, &env); err != nil {
		return fmt.Errorf("解析错误码诊断响应失败: %w", err)
	}
	if env.Code != 0 && env.Code != 200 {
		return fmt.Errorf("服务端错误: %s", env.Msg)
	}
	var card errorCodeCardResponse
	if err := json.Unmarshal(env.Data, &card); err != nil {
		return fmt.Errorf("解析 data 失败: %w", err)
	}
	if outputFormat == "json" {
		raw, _ := json.MarshalIndent(card, "", "  ")
		fmt.Println(string(raw))
		return nil
	}
	printErrorCodeCard(card)
	return nil
}

func printErrorCodeCard(c errorCodeCardResponse) {
	fmt.Printf("【错误码】 %s", c.Code)
	if c.Summary != "" {
		fmt.Printf("  — %s", c.Summary)
	}
	fmt.Println()
	if c.Source == "fallback_unknown" {
		fmt.Println("（未在目录命中，下面为占位回复）")
	}
	if rc := strings.TrimSpace(c.RootCause); rc != "" {
		fmt.Println("\n【根因】")
		fmt.Println(indentBlockTwoSpaces(rc))
	}
	if len(c.TypicalEvidence) > 0 {
		fmt.Println("\n【关键证据特征】")
		for _, e := range c.TypicalEvidence {
			fmt.Printf("  - %s\n", e)
		}
	}
	if rec := strings.TrimSpace(c.RecoveryOneLiner); rec != "" {
		fmt.Println("\n【立即恢复（在出问题节点 root 执行）】")
		fmt.Println(indentBlockTwoSpaces(rec))
	}
	if pf := strings.TrimSpace(c.PlatformFollowup); pf != "" {
		fmt.Println("\n【平台改进 / 已沉淀到代码】")
		fmt.Println(indentBlockTwoSpaces(pf))
	}
	if len(c.RelatedCodes) > 0 {
		fmt.Println("\n【关联错误码】")
		fmt.Printf("  %s\n", strings.Join(c.RelatedCodes, "  "))
	}
	if c.SkillName != "" {
		fmt.Printf("\n（来自技能 %s [%s]）\n", c.SkillName, c.SkillSource)
	}
}

// indentBlockTwoSpaces 与 node.go 中 indentBlock(s, prefix) 同名冲突，因此另起一个名字。
func indentBlockTwoSpaces(s string) string {
	lines := strings.Split(strings.TrimRight(s, "\n"), "\n")
	for i, ln := range lines {
		lines[i] = "  " + ln
	}
	return strings.Join(lines, "\n")
}

// opsfleetHTTPGet/Post: 轻量 HTTP 工具；与现有 callServerSkillsList 等保持一致行为（统一上下文超时 30s）。
func opsfleetHTTPGet(ctx context.Context, url string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode/100 != 2 {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return body, nil
}

func opsfleetHTTPPost(ctx context.Context, url string, payload []byte) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode/100 != 2 {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return body, nil
}
