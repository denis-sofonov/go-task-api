package auth

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

type RateLimiter struct {
	rdb    *redis.Client
	log    *slog.Logger
	limit  int64
	window time.Duration
}

func NewRateLimiter(rdb *redis.Client, log *slog.Logger, limit int64, window time.Duration) *RateLimiter {
	return &RateLimiter{rdb: rdb, log: log, limit: limit, window: window}
}

func (l *RateLimiter) Limit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			ip = r.RemoteAddr
		}
		key := "ratelimit:login:" + ip

		ctx, cancel := context.WithTimeout(r.Context(), 500*time.Millisecond)
		defer cancel()

		pipe := l.rdb.TxPipeline()
		count := pipe.Incr(ctx, key)
		pipe.ExpireNX(ctx, key, l.window)
		if _, err := pipe.Exec(ctx); err != nil {
			// Redis is unavailable: log it and let the request through.
			l.log.Error("rate limit: redis", "err", err)
			next.ServeHTTP(w, r)
			return
		}

		if count.Val() > l.limit {
			w.Header().Set("Retry-After", strconv.Itoa(int(l.window.Seconds())))
			writeError(w, http.StatusTooManyRequests, "too many login attempts")
			return
		}

		next.ServeHTTP(w, r)
	})
}
