package quota

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// usageFile is persisted state for daily LLM call counting.
type usageFile struct {
	Date  string `json:"date"` // YYYY-MM-DD local
	Count int    `json:"count"`
}

// DefaultCacheDir returns ~/.cache/ai-sre (or $XDG_CACHE_HOME/ai-sre).
func DefaultCacheDir() (string, error) {
	if d := os.Getenv("XDG_CACHE_HOME"); d != "" {
		return filepath.Join(d, "ai-sre"), nil
	}
	h, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(h, "ai-sre"), nil
}

// TakeDaily increments the daily counter and fails if maxPerDay > 0 and limit would be exceeded.
// maxPerDay == 0 means unlimited.
func TakeDaily(cacheDir string, maxPerDay int) error {
	if maxPerDay <= 0 {
		return nil
	}
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return fmt.Errorf("quota cache: %w", err)
	}
	path := filepath.Join(cacheDir, "llm_usage.json")
	today := time.Now().Format("2006-01-02")

	var u usageFile
	if b, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(b, &u)
	}
	if u.Date != today {
		u.Date = today
		u.Count = 0
	}
	if u.Count >= maxPerDay {
		return fmt.Errorf("已达到每日 LLM 调用上限 (%d 次)，可在 config.yaml 调整 max_llm_calls_per_day 或于次日重试", maxPerDay)
	}
	u.Count++
	b, err := json.MarshalIndent(u, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, b, 0o600); err != nil {
		return fmt.Errorf("quota write: %w", err)
	}
	return nil
}

// ReadUsage returns today's date string and LLM call count so far (no increment). For doctor / status.
func ReadUsage(cacheDir string) (date string, count int, err error) {
	path := filepath.Join(cacheDir, "llm_usage.json")
	b, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return time.Now().Format("2006-01-02"), 0, nil
		}
		return "", 0, err
	}
	var u usageFile
	if err := json.Unmarshal(b, &u); err != nil {
		return "", 0, err
	}
	today := time.Now().Format("2006-01-02")
	if u.Date != today {
		return today, 0, nil
	}
	return u.Date, u.Count, nil
}
