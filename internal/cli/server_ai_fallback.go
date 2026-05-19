package cli

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"strings"
	"syscall"
)

// serverAIFallbackEligible reports whether a server AI / OpsFleet API failure should
// trigger local LLM fallback. Auth, quota, and business errors are not eligible.
func serverAIFallbackEligible(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	if errors.Is(err, context.Canceled) {
		return false
	}
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}
	var op *net.OpError
	if errors.As(err, &op) {
		return true
	}
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		if urlErr.Timeout() {
			return true
		}
		return serverAIFallbackEligible(urlErr.Err)
	}
	var errno syscall.Errno
	if errors.As(err, &errno) {
		switch errno {
		case syscall.ECONNREFUSED, syscall.ECONNRESET, syscall.ENETUNREACH, syscall.EHOSTUNREACH, syscall.ETIMEDOUT:
			return true
		}
	}
	msg := strings.ToLower(err.Error())
	if serverAIAuthOrBusinessError(msg) {
		return false
	}
	if serverAIHTTPStatusEligible(msg) {
		return true
	}
	for _, hint := range []string{
		"connection refused",
		"connection reset",
		"no such host",
		"network is unreachable",
		"i/o timeout",
		"tls handshake timeout",
		"broken pipe",
		"dial tcp",
		"dial udp",
		"eof",
		"unexpected eof",
	} {
		if strings.Contains(msg, hint) {
			return true
		}
	}
	if strings.Contains(msg, "call server") {
		return strings.Contains(msg, "timeout") || strings.Contains(msg, "connection")
	}
	return false
}

func serverAIAuthOrBusinessError(lower string) bool {
	for _, s := range []string{
		"cli token",
		"token 无效",
		"token 已失效",
		"机器指纹",
		"机器不匹配",
		"未配置 opsfleet",
		"opsfleet_token",
		"能力不可执行",
		"paywall",
		"subscription",
		"quota",
		"额度",
		"status=401",
		"status=403",
		"status=402",
		"status=400",
		"status=404",
		"status=422",
		"api code=401",
		"api code=403",
		"api code=402",
	} {
		if strings.Contains(lower, s) {
			return true
		}
	}
	return false
}

func serverAIHTTPStatusEligible(lower string) bool {
	for _, code := range []string{"502", "503", "504", "520", "521", "522", "523", "524"} {
		if strings.Contains(lower, "status="+code) {
			return true
		}
	}
	return false
}

func notifyLocalAIFallback(reason error) {
	if reason == nil || os.Getenv("OPSFLEET_QUIET_LOCAL_FALLBACK") == "1" {
		return
	}
	_, _ = fmt.Fprintf(os.Stderr, "[%s] 服务端 AI 不可用，回退本机 LLM\n", progName)
}

func tryLocalAnalyzeDiagnose(ctx context.Context, topic string, kv map[string]string) (*diagnoseResponse, error) {
	eng, err := bootstrap()
	if err != nil {
		return nil, err
	}
	res, err := eng.Analyze(ctx, topic, kv, !noRAG)
	if err != nil {
		return nil, err
	}
	if res == nil || strings.TrimSpace(res.Answer) == "" {
		return nil, errors.New("本机 LLM 未返回有效诊断")
	}
	return &diagnoseResponse{
		Source:       "local",
		Answer:       res.Answer,
		SkillName:    res.SkillName,
		SkillDisplay: res.SkillDisplay,
	}, nil
}
