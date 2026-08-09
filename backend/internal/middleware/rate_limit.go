package middleware

import (
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

func RateLimit(redis *redis.Client) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID := GetUserID(r)
			key := rateLimitKey(userID)

			if !allowRequest(redis, key, w, r) {
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func IPRateLimit(redis *redis.Client) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := ipRateLimitKey(clientIP(r))

			if !allowRequest(redis, key, w, r) {
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func allowRequest(redis *redis.Client, key string, w http.ResponseWriter, r *http.Request) bool {
	count, err := redis.Incr(r.Context(), key).Result()
	if err != nil {
		// Fail-open: если Redis недоступен, не ломаем API.
		return true
	}

	if count == 1 {
		if err := redis.Expire(r.Context(), key, time.Minute).Err(); err != nil {
			return true
		}
	}

	if count > 100 {
		writeJSONError(w, http.StatusTooManyRequests, "too many requests")
		return false
	}

	return true
}

func rateLimitKey(userID int64) string {
	return "rate_limit:" + strconv.FormatInt(userID, 10)
}

func ipRateLimitKey(ip string) string {
	return "rate_limit:ip:" + ip
}

func clientIP(r *http.Request) string {
	if forwardedFor := r.Header.Get("X-Forwarded-For"); forwardedFor != "" {
		parts := strings.Split(forwardedFor, ",")
		return strings.TrimSpace(parts[0])
	}

	if realIP := r.Header.Get("X-Real-IP"); realIP != "" {
		return strings.TrimSpace(realIP)
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}

	return host
}
