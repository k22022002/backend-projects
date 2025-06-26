package system

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"task-tracker/cache"
	"task-tracker/common"

	"github.com/go-redis/redis/v8"
)

type RateLimitStatus struct {
	Remaining int   `json:"remaining"`
	Reset     int64 `json:"reset"` // seconds until reset
}

// RateLimitStatusHandler godoc
// @Summary Get current rate limit status
// @Description Return remaining requests and reset time
// @Tags Rate Limit
// @Produce json
// @Success 200 {object} system.RateLimitStatus
// @Failure 401 {string} string "Unauthorized"
// @Router /rate-limit [get]
func RateLimitStatusHandler(redisClient cache.RedisClientInterface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := r.Context().Value(common.ContextUserIDKey).(int)
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		ctx := context.Background()
		key := fmt.Sprintf("ratelimit:user:%d", userID)

		count := 0
		countVal, err := redisClient.Get(ctx, key).Result()
		if err != nil && !errors.Is(err, redis.Nil) {
			http.Error(w, "Failed to read rate limit", http.StatusInternalServerError)
			return
		}
		if err == nil {
			count, _ = strconv.Atoi(countVal)
		}

		ttl, err := redisClient.TTL(ctx, key).Result()
		if err != nil {
			http.Error(w, "Failed to read TTL", http.StatusInternalServerError)
			return
		}

		remaining := 100 - count
		if remaining < 0 {
			remaining = 0
		}

		resp := RateLimitStatus{
			Remaining: remaining,
			Reset:     int64(ttl.Seconds()),
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}
}
