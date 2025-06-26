package system

import (
	"context"
	"net/http"
	"net/http/httptest"
	"task-tracker/common"
	"task-tracker/middleware"
	"testing"
	"time"

	"github.com/go-redis/redis/v8"
)

func TestRateLimitMiddleware(t *testing.T) {
	rdb := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	defer rdb.Close()

	limiter := middleware.NewRateLimiter(rdb, 100, time.Hour)

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := limiter.Middleware(testHandler)

	req := httptest.NewRequest("GET", "/tasks", nil)
	ctx := context.WithValue(req.Context(), common.ContextUserIDKey, 123)

	req = req.WithContext(ctx)

	var lastStatus int
	for i := 0; i < 101; i++ {
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		lastStatus = rr.Code
	}

	if lastStatus != http.StatusTooManyRequests {
		t.Errorf("Expected 429 Too Many Requests, got %d", lastStatus)
	}
}
