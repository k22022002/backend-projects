package system

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"task-tracker/common"
)

func TestRateLimitStatusHandler(t *testing.T) {
	req := httptest.NewRequest("GET", "/rate-limit", nil)
	ctx := context.WithValue(req.Context(), common.ContextUserIDKey, 123)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	RateLimitStatusHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected 200 OK, got %d", rr.Code)
	}

	contentType := rr.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", contentType)
	}
}
