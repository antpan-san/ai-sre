package cli

import "testing"

func TestVersionIsOlder(t *testing.T) {
	cases := []struct {
		a, b     string
		older    bool
		describe string
	}{
		{"0.4.0", "0.4.1", true, "patch"},
		{"0.4.1", "0.4.1", false, "equal"},
		{"0.4.2", "0.4.1", false, "local newer"},
		{"0.3.9", "0.4.0", true, "minor"},
		{"v1.0.0", "1.0.1", true, "v prefix"},
	}
	for _, c := range cases {
		if got := versionIsOlder(c.a, c.b); got != c.older {
			t.Errorf("%s: versionIsOlder(%q,%q)=%v want %v", c.describe, c.a, c.b, got, c.older)
		}
	}
}
