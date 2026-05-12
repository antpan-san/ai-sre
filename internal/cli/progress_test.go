package cli

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestHumanBytes(t *testing.T) {
	cases := map[int64]string{
		0:                 "0B",
		1023:              "1023B",
		1024:              "1.0KB",
		1536:              "1.5KB",
		1024 * 1024:       "1.0MB",
		1024 * 1024 * 7:   "7.0MB",
		1024 * 1024 * 512: "512.0MB",
	}
	for in, want := range cases {
		if got := humanBytes(in); got != want {
			t.Errorf("humanBytes(%d) = %s, want %s", in, got, want)
		}
	}
}

func TestBuildBarBoundaries(t *testing.T) {
	if got := buildBar(0, 100, 8); got != "[--------]" {
		t.Errorf("zero filled = %q", got)
	}
	if got := buildBar(50, 100, 8); got != "[====----]" {
		t.Errorf("half filled = %q", got)
	}
	if got := buildBar(100, 100, 8); got != "[========]" {
		t.Errorf("full filled = %q", got)
	}
	// indeterminate (no total)
	if got := buildBar(50, 0, 8); !strings.HasPrefix(got, "[") || !strings.HasSuffix(got, "]") {
		t.Errorf("indeterminate bar = %q", got)
	}
}

func TestProgressReaderEmitsNonTTYLines(t *testing.T) {
	src := bytes.NewReader([]byte(strings.Repeat("x", 8192)))
	pr := &progressReader{
		r:         src,
		total:     8192,
		start:     time.Now().Add(-100 * time.Millisecond),
		out:       &bytes.Buffer{},
		tty:       false,
		minDrawDt: 0,
	}
	buf := make([]byte, 1024)
	for {
		_, err := pr.Read(buf)
		if err != nil {
			break
		}
	}
	_ = pr.Close()
	got := pr.out.(*bytes.Buffer).String()
	if !strings.Contains(got, "100.0%") {
		t.Errorf("expected 100%% line, got:\n%s", got)
	}
	if !strings.Contains(got, "8.0KB/8.0KB") {
		t.Errorf("expected 8.0KB/8.0KB summary, got:\n%s", got)
	}
}

func TestFormatDuration(t *testing.T) {
	if got := formatDuration(500 * time.Millisecond); got != "<1s" {
		t.Errorf("sub-second = %s", got)
	}
	if got := formatDuration(45 * time.Second); got != "45s" {
		t.Errorf("seconds = %s", got)
	}
	if got := formatDuration(90 * time.Second); got != "1m30s" {
		t.Errorf("minutes+seconds = %s", got)
	}
	if got := formatDuration(2*time.Hour + 15*time.Minute); got != "2h15m" {
		t.Errorf("hours+minutes = %s", got)
	}
}
