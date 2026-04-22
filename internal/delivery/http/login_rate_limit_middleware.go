package httpdelivery

import (
	"math"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	defaultLoginRateLimitMaxAttempts = 5
	defaultLoginRateLimitWindow      = 60 * time.Second
	messageTooManyLoginAttempts      = "too many login attempts, please try again later"
	errorCodeTooManyLoginAttempts    = "too_many_login_attempts"
)

type loginAttempt struct {
	count   int
	resetAt time.Time
}

type fixedWindowLoginLimiter struct {
	mu          sync.Mutex
	entries     map[string]loginAttempt
	maxAttempts int
	window      time.Duration
	lastCleanup time.Time
}

func newFixedWindowLoginLimiter(maxAttempts int, window time.Duration) *fixedWindowLoginLimiter {
	if maxAttempts <= 0 {
		maxAttempts = defaultLoginRateLimitMaxAttempts
	}
	if window <= 0 {
		window = defaultLoginRateLimitWindow
	}

	return &fixedWindowLoginLimiter{
		entries:     make(map[string]loginAttempt),
		maxAttempts: maxAttempts,
		window:      window,
	}
}

func (l *fixedWindowLoginLimiter) Allow(key string, now time.Time) (bool, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if key == "" {
		key = "unknown"
	}

	l.cleanupExpired(now)

	entry, exists := l.entries[key]
	if !exists || !now.Before(entry.resetAt) {
		l.entries[key] = loginAttempt{
			count:   1,
			resetAt: now.Add(l.window),
		}
		return true, 0
	}

	if entry.count >= l.maxAttempts {
		return false, entry.resetAt.Sub(now)
	}

	entry.count++
	l.entries[key] = entry

	return true, 0
}

func (l *fixedWindowLoginLimiter) cleanupExpired(now time.Time) {
	if !l.lastCleanup.IsZero() && now.Sub(l.lastCleanup) < l.window {
		return
	}

	for key, entry := range l.entries {
		if !now.Before(entry.resetAt) {
			delete(l.entries, key)
		}
	}

	l.lastCleanup = now
}

// NewLoginRateLimitMiddleware limits login requests per client IP in a fixed time window.
func NewLoginRateLimitMiddleware(maxAttempts int, window time.Duration) gin.HandlerFunc {
	limiter := newFixedWindowLoginLimiter(maxAttempts, window)

	return func(c *gin.Context) {
		clientIP := strings.TrimSpace(c.ClientIP())
		allowed, retryAfter := limiter.Allow(clientIP, time.Now())
		if allowed {
			c.Next()
			return
		}

		retryAfterSeconds := int(math.Ceil(retryAfter.Seconds()))
		if retryAfterSeconds < 1 {
			retryAfterSeconds = 1
		}

		c.Header("Retry-After", strconv.Itoa(retryAfterSeconds))
		AbortWithError(c, NewHTTPError(
			http.StatusTooManyRequests,
			messageTooManyLoginAttempts,
			gin.H{
				"code":                errorCodeTooManyLoginAttempts,
				"retry_after_seconds": retryAfterSeconds,
			},
		))
	}
}
