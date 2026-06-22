package user

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alexey-y-a/bank-api/internal/middleware"
	"github.com/alexey-y-a/bank-api/internal/repository/postgres"
	userservice "github.com/alexey-y-a/bank-api/internal/service/user"
	"github.com/alexey-y-a/bank-api/internal/test"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"
)

const testJWTSecret = "integration-test-secret-key-32b"

func setupIntegrationTest(t *testing.T) (*Handler, func()) {
	t.Helper()

	ctx := t.Context()

	dbHelper, err := test.NewPostgresHelper(ctx)
	require.NoError(t, err, "failed to start postgres container")

	err = dbHelper.RunMigrations("migrations")
	require.NoError(t, err, "failed to run migrations")

	userRepo := postgres.NewUserRepository(dbHelper.Pool)
	userSvc := userservice.NewService(userRepo, testJWTSecret, 24)
	handler := NewHandler(userSvc)

	cleanup := func() {
		_ = dbHelper.Cleanup()
	}

	return handler, cleanup
}

func TestHandler_Integration_Register(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	handler, cleanup := setupIntegrationTest(t)
	defer cleanup()

	tests := []struct {
		name           string
		body           interface{}
		expectedStatus int
		checkBody      func(t *testing.T, body map[string]interface{})
	}{
		{
			name: "успешная регистрация",
			body: RegisterRequest{
				Email:    "int-test@example.com",
				Username: "inttestuser",
				Password: "securepass123",
			},
			expectedStatus: http.StatusCreated,
			checkBody: func(t *testing.T, body map[string]interface{}) {
				require.Equal(t, "int-test@example.com", body["email"])
				require.Equal(t, "inttestuser", body["username"])
				require.NotEmpty(t, body["id"])
				require.NotEmpty(t, body["created_at"])
				_, hasPassword := body["password"]
				require.False(t, hasPassword, "password must not be in response")
			},
		},
		{
			name:           "ошибка: невалидный JSON",
			body:           "not-json",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "ошибка: дубликат email",
			body: RegisterRequest{
				Email:    "int-test@example.com",
				Username: "anotheruser",
				Password: "securepass123",
			},
			expectedStatus: http.StatusConflict,
		},
		{
			name: "ошибка: короткая валидация",
			body: RegisterRequest{
				Email:    "bad",
				Username: "ab",
				Password: "short",
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var bodyBytes []byte
			switch v := tt.body.(type) {
			case string:
				bodyBytes = []byte(v)
			default:
				bodyBytes, _ = json.Marshal(v)
			}

			req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			handler.Register(rec, req)

			require.Equal(t, tt.expectedStatus, rec.Code)

			if tt.checkBody != nil {
				var respBody map[string]interface{}
				err := json.Unmarshal(rec.Body.Bytes(), &respBody)
				require.NoError(t, err, "response should be valid JSON")
				tt.checkBody(t, respBody)
			}
		})
	}
}

func TestHandler_Integration_Login(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	handler, cleanup := setupIntegrationTest(t)
	defer cleanup()

	regBody, _ := json.Marshal(RegisterRequest{
		Email:    "login-test@example.com",
		Username: "loginuser",
		Password: "securepass123",
	})
	regReq := httptest.NewRequest(http.MethodPost, "/register", bytes.NewReader(regBody))
	regReq.Header.Set("Content-Type", "application/json")
	regRec := httptest.NewRecorder()
	handler.Register(regRec, regReq)
	require.Equal(t, http.StatusCreated, regRec.Code, "pre-registration must succeed")

	tests := []struct {
		name           string
		body           interface{}
		expectedStatus int
		checkToken     bool
	}{
		{
			name: "успешный логин",
			body: LoginRequest{
				Email:    "login-test@example.com",
				Password: "securepass123",
			},
			expectedStatus: http.StatusOK,
			checkToken:     true,
		},
		{
			name: "ошибка: неверный пароль",
			body: LoginRequest{
				Email:    "login-test@example",
				Password: "wrongpassword",
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "ошибкаЖ пользователь не найден",
			body: LoginRequest{
				Email:    "login-test@example",
				Password: "anypass",
			},
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bodyBytes, _ := json.Marshal(tt.body)
			req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			handler.Login(rec, req)

			require.Equal(t, tt.expectedStatus, rec.Code)

			if tt.checkToken {
				var resp LoginResponse
				err := json.Unmarshal(rec.Body.Bytes(), &resp)
				require.NoError(t, err, "response should be valid JSON")
				require.NotEmpty(t, resp.Token, "token must not be empty")
				require.Equal(t, "login-test@example.com", resp.User.Email)

				token, err := jwt.ParseWithClaims(resp.Token, &jwt.RegisteredClaims{}, func(t *jwt.Token) (interface{}, error) {
					return []byte(testJWTSecret), nil
				})
				require.NoError(t, err, "token must be valid JWT")
				require.True(t, token.Valid, "token must pass validation")

				claims, ok := token.Claims.(*jwt.RegisteredClaims)
				require.True(t, ok)
				require.NotEmpty(t, claims.Subject, "subject (userID) must be set")
			}

		})
	}
}

func TestHandler_Integration_AuthMiddleware(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	handler, cleanup := setupIntegrationTest(t)
	defer cleanup()

	regBody, _ := json.Marshal(RegisterRequest{
		Email:    "auth-mw-test@example.com",
		Username: "authmwuser",
		Password: "securepass123",
	})
	regReq := httptest.NewRequest(http.MethodPost, "/register", bytes.NewReader(regBody))
	regReq.Header.Set("Content-Type", "application/json")
	regRec := httptest.NewRecorder()
	handler.Register(regRec, regReq)
	require.Equal(t, http.StatusCreated, regRec.Code, "pre-registration must succeed")

	loginReq := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(regBody))
	loginReq.Header.Set("Content-Type", "application/json")
	loginRec := httptest.NewRecorder()
	handler.Login(loginRec, loginReq)
	require.Equal(t, http.StatusOK, loginRec.Code)

	var loginResp LoginResponse
	err := json.Unmarshal(loginRec.Body.Bytes(), &loginResp)
	require.NoError(t, err, "failed to parse login response")

	protectedHandler := middleware.Auth([]byte(testJWTSecret))(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID := middleware.GetUserID(r.Context())
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(userID))
		}),
	)

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+loginResp.Token)
	rec := httptest.NewRecorder()
	protectedHandler.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.NotEmpty(t, rec.Body.String(), "userID should be returned from context")

	reqNoAuth := httptest.NewRequest(http.MethodGet, "/protected", nil)
	recNoAuth := httptest.NewRecorder()
	protectedHandler.ServeHTTP(recNoAuth, reqNoAuth)
	require.Equal(t, http.StatusUnauthorized, recNoAuth.Code)

}
