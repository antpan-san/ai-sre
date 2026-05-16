package services

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"sync"
	"time"

	"ft-backend/common/config"
	"ft-backend/common/logger"

	"github.com/shirou/gopsutil/v4/disk"
)

var diskAlertState struct {
	mu       sync.Mutex
	lastSent time.Time
}

// RunDiskAlertMonitor periodically checks root disk usage and notifies DingTalk.
func RunDiskAlertMonitor(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			maybeNotifyDiskAlert()
		}
	}
}

func maybeNotifyDiskAlert() {
	cfg := config.ResolvedDiskAlertConfig()
	if !cfg.Enabled || cfg.Webhook == "" {
		return
	}
	pct, hostname, err := rootDiskUsagePercent()
	if err != nil {
		logger.Warn("disk alert sample: %v", err)
		return
	}
	if pct < cfg.ThresholdPercent {
		return
	}
	diskAlertState.mu.Lock()
	if !diskAlertState.lastSent.IsZero() && time.Since(diskAlertState.lastSent) < cfg.Cooldown {
		diskAlertState.mu.Unlock()
		return
	}
	diskAlertState.lastSent = time.Now()
	diskAlertState.mu.Unlock()

	title := "【磁盘告警】控制台主机"
	body := fmt.Sprintf("主机: %s\n根分区使用率: %.1f%%\n阈值: %.0f%%\n时间: %s",
		hostname, pct, cfg.ThresholdPercent, time.Now().Format("2006-01-02 15:04:05"))
	if err := SendDingTalkText(cfg.Webhook, cfg.Keyword, title, body); err != nil {
		logger.Warn("disk alert dingtalk: %v", err)
	}
}

func rootDiskUsagePercent() (float64, string, error) {
	path := "/"
	if runtime.GOOS == "windows" {
		path = `C:\`
	}
	usage, err := disk.Usage(path)
	if err != nil {
		return 0, "", err
	}
	host, _ := os.Hostname()
	if host == "" {
		host = "unknown"
	}
	return usage.UsedPercent, host, nil
}
