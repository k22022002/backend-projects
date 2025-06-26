package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"task-tracker/cache"
	"time"

	"github.com/go-redis/redis/v8"
)

type RateLimiter struct {
	RedisClient cache.RedisClientInterface
	Limit       int
	Window      time.Duration
}

func NewRateLimiter(redisClient cache.RedisClientInterface, limit int, window time.Duration) *RateLimiter {
	return &RateLimiter{
		RedisClient: redisClient,
		Limit:       limit,
		Window:      window,
	}
}

func (r *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		ctx := context.Background()
		userID := req.Context().Value("user_id")
		if userID == nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		key := fmt.Sprintf("rate_limit:%v", userID)
		remainingKey := key + ":remaining"
		resetKey := key + ":reset"

		remaining, err := r.RedisClient.Get(ctx, remainingKey).Int()
		if err == redis.Nil {
			r.RedisClient.Set(ctx, remainingKey, r.Limit-1, r.Window)
			reset := time.Now().Add(r.Window).Unix()
			r.RedisClient.Set(ctx, resetKey, reset, r.Window)

			w.Header().Set("X-Rate-Limit-Remaining", strconv.Itoa(r.Limit-1))
			w.Header().Set("X-Rate-Limit-Reset", strconv.FormatInt(reset, 10))
			next.ServeHTTP(w, req)
			return
		} else if err != nil {
			http.Error(w, "Rate limit error", http.StatusInternalServerError)
			return
		}

		if remaining <= 0 {
			reset, _ := r.RedisClient.Get(ctx, resetKey).Result()
			w.Header().Set("X-Rate-Limit-Remaining", "0")
			w.Header().Set("X-Rate-Limit-Reset", reset)
			http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
			return
		}

		r.RedisClient.Decr(ctx, remainingKey)
		reset, _ := r.RedisClient.Get(ctx, resetKey).Result()
		w.Header().Set("X-Rate-Limit-Remaining", strconv.Itoa(remaining-1))
		w.Header().Set("X-Rate-Limit-Reset", reset)
		next.ServeHTTP(w, req)
	})
}
