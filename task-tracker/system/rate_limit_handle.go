package system

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"task-tracker/cache"
	"task-tracker/common"
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
func RateLimitStatusHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(common.ContextUserIDKey).(int)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	ctx := context.Background()
	key := fmt.Sprintf("ratelimit:user:%d", userID)

	count, err := cache.RedisClient.Get(ctx, key).Int()
	if err != nil && err.Error() != "redis: nil" {
		http.Error(w, "Failed to read rate limit", http.StatusInternalServerError)
		return
	}

	ttl, err := cache.RedisClient.TTL(ctx, key).Result()
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
