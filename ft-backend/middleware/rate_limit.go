package middleware

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// rateLimitClientKey 在反代（Nginx/Vite）后 RemoteAddr 常为 127.0.0.1 时，用 X-Forwarded-For 首段区分真实客户端，
// 避免全站用户共享同一限流桶导致「一直登录拒绝」。
func rateLimitClientKey(c *gin.Context) string {
	host, _, err := net.SplitHostPort(c.Request.RemoteAddr)
	if err != nil {
		host = c.Request.RemoteAddr
		host = strings.TrimPrefix(strings.TrimSuffix(host, "]"), "[")
	}
	ip := net.ParseIP(host)
	if ip != nil && ip.IsLoopback() {
		if xff := strings.TrimSpace(c.GetHeader("X-Forwarded-For")); xff != "" {
			parts := strings.Split(xff, ",")
			if len(parts) > 0 {
				client := strings.TrimSpace(parts[0])
				if client != "" {
					return client
				}
			}
		}
	}
	return host
}

type rateEntry struct {
	count     int
	resetTime time.Time
}

type rateStore struct {
	mu      sync.Mutex
	entries map[string]rateEntry
}

func newRateStore() *rateStore {
	return &rateStore{entries: make(map[string]rateEntry)}
}

func (s *rateStore) allow(key string, limit int, window time.Duration, now time.Time) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	entry := s.entries[key]
	if entry.resetTime.IsZero() || now.After(entry.resetTime) {
		s.entries[key] = rateEntry{count: 1, resetTime: now.Add(window)}
		return true
	}
	if entry.count >= limit {
		return false
	}
	entry.count++
	s.entries[key] = entry

	if len(s.entries) > 10000 {
		for k, v := range s.entries {
			if now.After(v.resetTime) {
				delete(s.entries, k)
			}
		}
	}
	return true
}

// RateLimit is a lightweight fixed-window limiter intended for public abuse
// control. It is process-local and intentionally avoids network round trips.
func RateLimit(name string, limit int, window time.Duration) gin.HandlerFunc {
	store := newRateStore()
	return func(c *gin.Context) {
		key := name + ":" + rateLimitClientKey(c)
		if !store.allow(key, limit, window, time.Now()) {
			c.JSON(http.StatusTooManyRequests, gin.H{"code": 429, "msg": "请求过于频繁，请稍后再试"})
			c.Abort()
			return
		}
		c.Next()
	}
}
