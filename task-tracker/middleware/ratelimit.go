package middleware

import (
	"fmt"
	"net/http"
	"strconv"
	"task-tracker/cache"
	"task-tracker/common"
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
		ctx := req.Context() // ✅ Dùng context gốc từ request

		val := ctx.Value(common.ContextUserIDKey)
		userID, ok := val.(int)
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		key := fmt.Sprintf("rate_limit:%d", userID)
		remainingKey := key + ":remaining"
		resetKey := key + ":reset"

		remaining, err := r.RedisClient.Get(ctx, remainingKey).Int()
		if err == redis.Nil {
			// Khởi tạo lần đầu
			err1 := r.RedisClient.Set(ctx, remainingKey, r.Limit-1, r.Window).Err()
			err2 := r.RedisClient.Set(ctx, resetKey, time.Now().Add(r.Window).Unix(), r.Window).Err()

			if err1 != nil || err2 != nil {
				fmt.Printf("[DEBUG] Redis SET error: %v | %v\n", err1, err2)
				http.Error(w, "Rate limit setup error", http.StatusInternalServerError)
				return
			}

			w.Header().Set("X-Rate-Limit-Remaining", strconv.Itoa(r.Limit-1))
			w.Header().Set("X-Rate-Limit-Reset", strconv.FormatInt(time.Now().Add(r.Window).Unix(), 10))
			next.ServeHTTP(w, req)
			return
		} else if err != nil {
			fmt.Printf("[DEBUG] Redis GET error: %v\n", err)
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

		if err := r.RedisClient.Decr(ctx, remainingKey).Err(); err != nil {
			fmt.Printf("[DEBUG] Redis DECR error: %v\n", err)
			http.Error(w, "Rate limit update error", http.StatusInternalServerError)
			return
		}

		reset, _ := r.RedisClient.Get(ctx, resetKey).Result()
		w.Header().Set("X-Rate-Limit-Remaining", strconv.Itoa(remaining-1))
		w.Header().Set("X-Rate-Limit-Reset", reset)
		next.ServeHTTP(w, req)
	})
}
