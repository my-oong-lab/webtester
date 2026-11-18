package middleware

import (
	"net/http"
	"strings"
	"time"

	"webtester/internal/auth"
	apperr "webtester/pkg/errors"
	"webtester/pkg/response"
)

// RateLimitConfig holds rate limiting configuration.
type RateLimitConfig struct {
	Limit  int
	Window time.Duration
}

// RateLimit returns a middleware that applies rate limiting based on client IP.
func RateLimit(redisStore *auth.RedisStore, cfg RateLimitConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := clientIP(r)
			key := "ratelimit:" + ip
			allowed, err := redisStore.RateLimit(r.Context(), key, cfg.Limit, cfg.Window)
			if err != nil {
				response.Fail(w, apperr.Internal("rate limit check failed", err))
				return
			}
			if !allowed {
				response.Fail(w, apperr.RateLimited("too many requests"))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// clientIP extracts the client IP from the request.
func clientIP(r *http.Request) string {
	// Check X-Forwarded-For header
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		ips := strings.Split(xff, ",")
		return strings.TrimSpace(ips[0])
	}
	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	// Fall back to RemoteAddr
	return r.RemoteAddr
}