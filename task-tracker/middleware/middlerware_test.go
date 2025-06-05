package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"task-tracker/middleware"
	"task-tracker/system"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
)

func generateTestToken(userID int) string {
	claims := &system.Claims{
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
		extractedUserID = r.Context().Value("userID").(int)
		w.WriteHeader(http.StatusOK)
	})

	middleware.JWTMiddleware(next).ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, 123, extractedUserID)
}
