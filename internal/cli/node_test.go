package cli

import (
	"strings"
	"testing"
)

// 时区任务必须在 `meta: end_play` 之前——否则当目标已有 NTP 服务且
// --on-conflict=skip 时，时区设置会被一并跳过（0.4.6 复现的实际事故）。
func TestTimeSyncPlaybook_TimezoneRunsBeforeSkipEndPlay(t *testing.T) {
	pb := genTimeSyncPlaybook(timeSyncPlaybookOpts{
		Timezone:   "Asia/Shanghai",
		Tool:       "chrony",
		OnConflict: "skip",
		NTPTarget:  "ntp.aliyun.com",
	})
	tzIdx := strings.Index(pb, "设置时区（始终先于 NTP 处理执行）")
	endIdx := strings.Index(pb, "meta: end_play")
	if tzIdx < 0 {
		t.Fatalf("timezone task missing from playbook:\n%s", pb)
	}
	if endIdx < 0 {
		t.Fatalf("expected meta: end_play in skip playbook:\n%s", pb)
	}
	if tzIdx > endIdx {
		t.Fatalf("timezone task must precede end_play; tzIdx=%d endIdx=%d\n%s", tzIdx, endIdx, pb)
	}
	if !strings.Contains(pb, `timezone: { name: "Asia/Shanghai" }`) {
		t.Fatalf("timezone scalar must be properly double-quoted; got:\n%s", pb)
	}
}

// 没有任何 `when:` 表达式以双引号开头——0.4.5 因为把
// `when: "chrony" == "chrony"` 喂给 Ansible 而炸 YAML 解析器。
func TestTimeSyncPlaybook_NoLeadingDoubleQuoteInWhen(t *testing.T) {
	cases := []timeSyncPlaybookOpts{
		{Timezone: "Asia/Shanghai", Tool: "chrony", OnConflict: "skip", NTPTarget: "ntp.aliyun.com"},
		{Timezone: "Asia/Shanghai", Tool: "chrony", OnConflict: "force", NTPTarget: "ntp.aliyun.com"},
		{Timezone: "Asia/Shanghai", Tool: "timesyncd", OnConflict: "skip", NTPTarget: "ntp.aliyun.com", Fallback: "ntp1.aliyun.com", SyncIntervalMin: 15},
		{IsSelfHosted: true, Timezone: "Asia/Shanghai", Tool: "chrony", OnConflict: "force", NTPTarget: "192.168.56.10"},
	}
	for i, c := range cases {
		pb := genTimeSyncPlaybook(c)
		for _, line := range strings.Split(pb, "\n") {
			s := strings.TrimSpace(line)
			if !strings.HasPrefix(s, "when:") {
				continue
			}
			expr := strings.TrimSpace(strings.TrimPrefix(s, "when:"))
			if strings.HasPrefix(expr, `"`) {
				t.Errorf("case %d: when expression must not start with double-quote (Ansible YAML scalar trap): %q\n%s", i, expr, pb)
			}
		}
	}
}

// sys-param 同样不允许出现以双引号开头的 when 表达式。
func TestSysParamPlaybook_NoLeadingDoubleQuoteInWhen(t *testing.T) {
	cases := []sysParamPlaybookOpts{
		{Rows: defaultSysctlRows(), OnConflict: "skip", DisableSwap: true, RaiseUlimit: true},
		{Rows: map[string]string{"vm.swappiness": "5"}, OnConflict: "force", DisableSwap: false, RaiseUlimit: false},
	}
	for i, c := range cases {
		pb := genSysParamPlaybook(c)
		for _, line := range strings.Split(pb, "\n") {
			s := strings.TrimSpace(line)
			if !strings.HasPrefix(s, "when:") {
				continue
			}
			expr := strings.TrimSpace(strings.TrimPrefix(s, "when:"))
			if strings.HasPrefix(expr, `"`) {
				t.Errorf("case %d: when expression must not start with double-quote: %q\n%s", i, expr, pb)
			}
		}
	}
}
