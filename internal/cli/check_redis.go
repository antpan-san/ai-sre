package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

func finishRedisCheckEvidence(topic string, ctx map[string]string) error {
	if normalizeCheckTopic(topic) != "redis" {
		return nil
	}
	if strings.TrimSpace(ctx["redis_diagnose_json"]) != "" && ctx["redis_auth_required"] != "true" {
		return nil
	}
	body, report, err := collectRedisProbeJSON(nil, ctx)
	if err == errRedisAuthRequired {
		if strings.EqualFold(outputFormat, "json") {
			payload := map[string]any{"auth_required": true, "address": report.Address}
			_ = json.Unmarshal([]byte(body), &payload)
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(payload)
		}
		return fmt.Errorf("Redis %s 需要密码认证；请在本机终端重试以交互输入，或使用 -d password=（不会上传服务端）", report.Address)
	}
	if err != nil {
		return err
	}
	if body != "" {
		ctx["redis_diagnose_json"] = body
	}
	delete(ctx, "redis_auth_required")
	return nil
}

func stripSensitiveCheckContext(ctx map[string]string) {
	if ctx == nil {
		return
	}
	delete(ctx, "password")
}
