package middleware_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"task-tracker/common"
	"task-tracker/middleware"
	"testing"
	"time"

	"github.com/alicebob/miniredis"
	"github.com/go-redis/redis/v8"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func generateTestToken(userID int) string {
	claims := &common.Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, _ := token.SignedString([]byte("your-secret-key"))
	return signedToken
}

func TestJWTMiddleware_NoAuthorizationHeader(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()

	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})

	middleware.JWTMiddleware(next).ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.False(t, called)
}

func TestJWTMiddleware_InvalidFormat(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Token abc123")
	rec := httptest.NewRecorder()

	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})

	middleware.JWTMiddleware(next).ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.False(t, called)
}

func TestJWTMiddleware_InvalidToken(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer invalid.token.value")
	rec := httptest.NewRecorder()

	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})

	middleware.JWTMiddleware(next).ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.False(t, called)
}

func TestJWTMiddleware_ValidToken(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	token := generateTestToken(123)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	var extractedUserID int
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		val := r.Context().Value(common.ContextUserIDKey)
		userID, ok := val.(int)
		if !ok {
			t.Fatal("user_id not found or invalid")
		}
		extractedUserID = userID

		w.WriteHeader(http.StatusOK)
	})

	middleware.JWTMiddleware(next).ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, 123, extractedUserID)
}
func TestRateLimitMiddleware(t *testing.T) {
	s, err := miniredis.Run()
	require.NoError(t, err)
	defer s.Close()

	rdb := redis.NewClient(&redis.Options{
		Addr: s.Addr(),
	})

	limiter := middleware.NewRateLimiter(rdb, 100, time.Hour)

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := limiter.Middleware(testHandler)

	req := httptest.NewRequest("GET", "/tasks", nil)
	ctx := context.WithValue(req.Context(), "user_id", "testuser") // dùng string để dễ gán key
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
