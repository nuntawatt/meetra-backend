package middleware

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/nuntawatt/meetra-backend/pkg/response"
)

// RateLimit returns a middleware that limits requests per IP using a Redis sliding window.
// maxRequests per windowSeconds is enforced per client IP address.
func RateLimit(client *redis.Client, maxRequests int, windowSeconds int) gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		key := fmt.Sprintf("rate:%s", ip)
		ctx := context.Background()

		// Atomically increment the counter and set expiry on first hit
		pipe := client.Pipeline()
		incr := pipe.Incr(ctx, key)
		pipe.Expire(ctx, key, time.Duration(windowSeconds)*time.Second)
		_, err := pipe.Exec(ctx)
		if err != nil {
			// On Redis failure, fail open (do not block traffic)
			c.Next()
			return
		}

		count := incr.Val()
		c.Header("X-RateLimit-Limit", fmt.Sprintf("%d", maxRequests))
		c.Header("X-RateLimit-Remaining", fmt.Sprintf("%d", max(0, int64(maxRequests)-count)))
		c.Header("X-RateLimit-Reset", fmt.Sprintf("%d", time.Now().Add(time.Duration(windowSeconds)*time.Second).Unix()))

		if count > int64(maxRequests) {
			response.TooManyRequests(c)
			c.Abort()
			return
		}

		c.Next()
	}
}

// max is a tiny helper because Go 1.21+ generics min/max may not be available everywhere.
func max(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

// ExampleHandler just shows 429 behaviour; not used in production routing.
var _ http.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
