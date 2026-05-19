package cli

import "testing"

func TestFormatCheckAnswerTextPlain(t *testing.T) {
	in := "## 根因与触发条件\n**碎片**\n## 关键指标证据\n- x"
	out := formatCheckAnswerText("redis", in)
	if out == "" || containsSubstring(out, "##") {
		t.Fatalf("got %q", out)
	}
	if !containsSubstring(out, "【根因与触发条件】") {
		t.Fatalf("want bracket title, got %q", out)
	}
}

func containsSubstring(s, sub string) bool {
	return len(sub) == 0 || (len(s) >= len(sub) && indexSubstring(s, sub) >= 0)
}

func indexSubstring(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
