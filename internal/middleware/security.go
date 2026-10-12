package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// SecureHeaders sets the same baseline headers Express's helmet() applies
// by default (src/index.ts: app.use(helmet())).
func SecureHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		h := c.Writer.Header()
		h.Set("X-DNS-Prefetch-Control", "off")
		h.Set("X-Frame-Options", "SAMEORIGIN")
		h.Set("Strict-Transport-Security", "max-age=15552000; includeSubDomains")
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Download-Options", "noopen")
		h.Set("X-Permitted-Cross-Domain-Policies", "none")
		h.Set("Referrer-Policy", "no-referrer")
		c.Next()
	}
}

// RateLimiter mirrors express-rate-limit's defaults used in src/index.ts:
// 100 requests per IP per 15-minute fixed window.
func RateLimiter(max int, window time.Duration) gin.HandlerFunc {
	type bucket struct {
		count   int
		resetAt time.Time
	}

	var (
		mu      sync.Mutex
		buckets = map[string]*bucket{}
	)

	return func(c *gin.Context) {
		ip := c.ClientIP()
		now := time.Now()

		mu.Lock()
		b, ok := buckets[ip]
		if !ok || now.After(b.resetAt) {
			b = &bucket{count: 0, resetAt: now.Add(window)}
			buckets[ip] = b
		}
		b.count++
		exceeded := b.count > max
		mu.Unlock()

		if exceeded {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"message": "Too many requests to the ip, please try again later.",
			})
			return
		}
		c.Next()
	}
}
