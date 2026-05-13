package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

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
		key := name + ":" + c.ClientIP()
		if !store.allow(key, limit, window, time.Now()) {
			c.JSON(http.StatusTooManyRequests, gin.H{"code": 429, "msg": "请求过于频繁，请稍后再试"})
			c.Abort()
			return
		}
		c.Next()
	}
}
