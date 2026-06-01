package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"
)

var testSecret = []byte("test-secret-key-for-unit-tests")

func generateTestToken(userID string, expiresAt time.Time) string {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Subject:   userID,
		ExpiresAt: jwt.NewNumericDate(expiresAt),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
	})
	tokenString, _ := token.SignedString(testSecret)
	return tokenString
}

func TestAuth(t *testing.T) {
	tests := []struct {
		name           string
		authHeader     string
		expectedStatus int
		expectedUserID string
	}{
		{
			name:           "валидный токен",
			authHeader:     "Bearer " + generateTestToken("user-123", time.Now().Add(1*time.Hour)),
			expectedStatus: http.StatusOK,
			expectedUserID: "user-123",
		},
		{
			name:           "просроченный токен",
			authHeader:     "Bearer " + generateTestToken("user-456", time.Now().Add(-1*time.Hour)),
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "отсутствующий заголовок Authorization",
			authHeader:     "",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "некорректный формат: нет Bearer prefix",
			authHeader:     "Basic some-token",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "некорректный формат: пустой токен после Bearer",
			authHeader:     "Bearer ",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "неверная подпись",
			authHeader:     "Bearer " + generateTestTokenWithWrongSecret("user-789"),
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nextCalled := false
			var capturedUserID string

			nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				nextCalled = true
				capturedUserID = GetUserID(r.Context())
				w.WriteHeader(http.StatusOK)
			})

			handler := Auth(testSecret)(nextHandler)

			req := httptest.NewRequest(http.MethodGet, "/protected", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}

			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			require.Equal(t, tt.expectedStatus, rec.Code)

			if tt.expectedUserID != "" {
				require.True(t, nextCalled, "next handler должен быть вызван при валидном токене")
				require.Equal(t, tt.expectedUserID, capturedUserID, "userID в контексте должен совпадать")
			} else {
				require.False(t, nextCalled, "next handler НЕ должен быть вызван при ошибке аутентификации")
			}
		})
	}
}

func generateTestTokenWithWrongSecret(userID string) string {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Subject:   userID,
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
	})
	wrongSecret := []byte("completely-wrong-secret")
	tokenString, _ := token.SignedString(wrongSecret)
	return tokenString
}
