package middleware

import (
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type rateLimitEntry struct {
	count     int
	windowEnd time.Time
}

type ipRateLimiter struct {
	mu          sync.Mutex
	entries     map[string]rateLimitEntry
	limit       int
	window      time.Duration
	lastCleanup time.Time
}

// RateLimit 为单实例提供轻量的按客户端 IP 固定窗口限流。
// 它用于减少登录爆破和公开写接口被滥用；多实例部署时应再由网关统一限流。
func RateLimit(limit int, window time.Duration) gin.HandlerFunc {
	limiter := &ipRateLimiter{
		entries:     make(map[string]rateLimitEntry),
		limit:       limit,
		window:      window,
		lastCleanup: time.Now(),
	}

	return func(c *gin.Context) {
		allowed, retryAfter := limiter.allow(c.ClientIP(), time.Now())
		if !allowed {
			c.Header("Retry-After", strconv.Itoa(max(1, int(retryAfter.Seconds()))))
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "请求过于频繁，请稍后再试",
			})
			return
		}
		c.Next()
	}
}

func (l *ipRateLimiter) allow(key string, now time.Time) (bool, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if now.Sub(l.lastCleanup) >= l.window {
		for entryKey, entry := range l.entries {
			if !now.Before(entry.windowEnd) {
				delete(l.entries, entryKey)
			}
		}
		l.lastCleanup = now
	}

	entry, exists := l.entries[key]
	if !exists || !now.Before(entry.windowEnd) {
		l.entries[key] = rateLimitEntry{count: 1, windowEnd: now.Add(l.window)}
		return true, 0
	}
	if entry.count >= l.limit {
		return false, time.Until(entry.windowEnd)
	}

	entry.count++
	l.entries[key] = entry
	return true, 0
}
