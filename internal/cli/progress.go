package cli

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync/atomic"
	"time"
)

// progressReader wraps an io.Reader and emits a CLI-style progress bar to stderr
// while data flows through. It is TTY-aware: when stderr is not a TTY (e.g.
// piped to ansible / journal / a log file) only periodic line summaries are
// printed so logs stay readable.
//
// Usage:
//
//	pr := newProgressReader(resp.Body, resp.ContentLength, "下载 ai-sre 二进制")
//	defer pr.Close()
//	if _, err := io.Copy(out, pr); err != nil { ... }
type progressReader struct {
	r         io.Reader
	total     int64
	label     string
	written   int64
	start     time.Time
	lastDraw  time.Time
	out       io.Writer
	tty       bool
	closed    atomic.Bool
	minDrawDt time.Duration
}

func newProgressReader(r io.Reader, total int64, label string) *progressReader {
	pr := &progressReader{
		r:         r,
		total:     total,
		label:     strings.TrimSpace(label),
		start:     time.Now(),
		out:       os.Stderr,
		tty:       isStderrTTY(),
		minDrawDt: 120 * time.Millisecond,
	}
	if !pr.tty {
		// non-TTY: throttle to once per second to keep logs readable
		pr.minDrawDt = time.Second
	}
	if pr.label != "" {
		if pr.tty {
			fmt.Fprintf(pr.out, "%s ...\n", pr.label)
		} else {
			fmt.Fprintf(pr.out, "[%s] %s ...\n", time.Now().Format("15:04:05"), pr.label)
		}
	}
	return pr
}

func (p *progressReader) Read(buf []byte) (int, error) {
	n, err := p.r.Read(buf)
	if n > 0 {
		newWritten := atomic.AddInt64(&p.written, int64(n))
		now := time.Now()
		if now.Sub(p.lastDraw) >= p.minDrawDt || err == io.EOF {
			p.draw(newWritten, false)
			p.lastDraw = now
		}
	}
	if err == io.EOF {
		p.draw(atomic.LoadInt64(&p.written), true)
	}
	return n, err
}

// Close finalises the progress line (forces a final draw).
func (p *progressReader) Close() error {
	if !p.closed.CompareAndSwap(false, true) {
		return nil
	}
	p.draw(atomic.LoadInt64(&p.written), true)
	return nil
}

func (p *progressReader) draw(written int64, final bool) {
	if p.out == nil {
		return
	}
	elapsed := time.Since(p.start)
	if elapsed <= 0 {
		elapsed = time.Millisecond
	}
	bytesPerSec := float64(written) / elapsed.Seconds()
	percentStr := "—"
	etaStr := ""
	if p.total > 0 {
		pct := float64(written) * 100 / float64(p.total)
		if pct > 100 {
			pct = 100
		}
		percentStr = fmt.Sprintf("%5.1f%%", pct)
		if bytesPerSec > 0 && written < p.total {
			remain := float64(p.total-written) / bytesPerSec
			etaStr = " eta " + formatDuration(time.Duration(remain*float64(time.Second)))
		}
	}
	if p.tty {
		bar := buildBar(written, p.total, 24)
		line := fmt.Sprintf("\r  %s %s %s/%s %s/s%s",
			bar, percentStr, humanBytes(written), totalStrOrUnknown(p.total),
			humanBytes(int64(bytesPerSec)), etaStr,
		)
		fmt.Fprint(p.out, line)
		if final {
			fmt.Fprint(p.out, "\n")
		}
	} else {
		// non-TTY: print one line per draw
		fmt.Fprintf(p.out, "[%s] %s %s/%s %s/s%s\n",
			time.Now().Format("15:04:05"), percentStr,
			humanBytes(written), totalStrOrUnknown(p.total),
			humanBytes(int64(bytesPerSec)), etaStr,
		)
	}
}

func buildBar(done, total int64, width int) string {
	if total <= 0 {
		// indeterminate: spinner of static length
		return "[" + strings.Repeat(".", width) + "]"
	}
	filled := int(float64(width) * float64(done) / float64(total))
	if filled > width {
		filled = width
	}
	return "[" + strings.Repeat("=", filled) + strings.Repeat("-", width-filled) + "]"
}

func humanBytes(n int64) string {
	if n < 0 {
		return "0B"
	}
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%dB", n)
	}
	div, exp := int64(unit), 0
	for x := n / unit; x >= unit; x /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f%sB", float64(n)/float64(div), [...]string{"K", "M", "G", "T", "P"}[exp])
}

func totalStrOrUnknown(total int64) string {
	if total <= 0 {
		return "?"
	}
	return humanBytes(total)
}

func formatDuration(d time.Duration) string {
	if d < time.Second {
		return "<1s"
	}
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm%02ds", int(d.Minutes()), int(d.Seconds())%60)
	}
	return fmt.Sprintf("%dh%02dm", int(d.Hours()), int(d.Minutes())%60)
}

// isStderrTTY reports whether stderr is attached to a terminal.
// Avoids pulling in golang.org/x/term to keep deps tiny; uses Stat + ModeCharDevice.
func isStderrTTY() bool {
	if os.Getenv("OPSFLEET_NO_PROGRESS") == "1" {
		return false
	}
	fi, err := os.Stderr.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}

func isStdinTTY() bool {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}
