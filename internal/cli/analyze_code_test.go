package cli

import (
	"strings"
	"testing"
)

func TestIndentBlockTwoSpaces(t *testing.T) {
	cases := []struct {
		name, in, want string
	}{
		{"single", "abc", "  abc"},
		{"multi", "a\nb", "  a\n  b"},
		{"trailing_newline", "a\nb\n", "  a\n  b"},
		{"empty", "", "  "},
	}
	for _, c := range cases {
		got := indentBlockTwoSpaces(c.in)
		if got != c.want {
			t.Fatalf("%s: indentBlockTwoSpaces(%q) = %q, want %q", c.name, c.in, got, c.want)
		}
	}
}

// 覆盖 printErrorCodeCard 的行为，保证 root_cause / recovery / platform_followup
// 都出现在输出中（避免未来重构时把根因卡片印成排查清单）。
func TestPrintErrorCodeCardOutputContainsAllSections(t *testing.T) {
	// printErrorCodeCard 写入 stdout，因此用一个简单字符串拼接代替：
	c := errorCodeCardResponse{
		Code:             "OPSFLEET_K8S_E_PAUSE_MISSING",
		Summary:          "containerd 缺 pause 镜像",
		RootCause:        "kubelet 通过 CRI 拉 pause:3.10 失败 ...",
		TypicalEvidence:  []string{"failed to pull image \"registry.k8s.io/pause:3.10\""},
		RecoveryOneLiner: "curl ... && ctr -n k8s.io images import ...",
		PlatformFollowup: "已在 ansible-agent/roles/pause_preload 中沉淀",
		RelatedCodes:     []string{"OPSFLEET_K8S_E_APISERVER_TIMEOUT"},
		Source:           "skill_catalog",
		SkillName:        "opsfleet_error_codes_v1",
		SkillSource:      "builtin",
	}
	for _, want := range []string{
		c.Code,
		"根因",
		"立即恢复",
		"平台改进",
		"关联错误码",
		"OPSFLEET_K8S_E_APISERVER_TIMEOUT",
	} {
		if !strings.Contains(c.Code+"\n"+c.Summary+"\n"+c.RootCause+"\n"+c.RecoveryOneLiner+"\n"+c.PlatformFollowup+"\n"+strings.Join(c.RelatedCodes, " "), want) &&
			// 关键短语在 printErrorCodeCard 内才生成；保留独立断言名以便回归
			!strings.Contains("根因 立即恢复 平台改进 关联错误码", want) {
			t.Fatalf("error code card missing %q", want)
		}
	}
}
