package user

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alexey-y-a/bank-api/internal/domain"
	userservice "github.com/alexey-y-a/bank-api/internal/service/user"
	"github.com/stretchr/testify/require"
)

type mockUserService struct {
	registerFn func(ctx context.Context, email, username, password string) (*domain.User, error)
	loginFn    func(ctx context.Context, email, password string) (string, *domain.User, error)
}

func (m *mockUserService) Register(ctx context.Context, email, username, password string) (*domain.User, error) {
	return m.registerFn(ctx, email, username, password)
}

func (m *mockUserService) Login(ctx context.Context, email, password string) (string, *domain.User, error) {
	return m.loginFn(ctx, email, password)
}

func TestHandler_Register(t *testing.T) {
	tests := []struct {
		name           string
		body           interface{}
		setupMock      func(m *mockUserService)
		expectedStatus int
		expectedBody   string
	}{
		{
			name: "успешная регистрация",
			body: RegisterRequest{
				Email:    "new@example.com",
				Username: "newuser",
				Password: "securepass123",
			},
			setupMock: func(m *mockUserService) {
				m.registerFn = func(ctx context.Context, email, username, password string) (*domain.User, error) {
					return &domain.User{ID: 1, Email: email, Username: username, Password: password}, nil
				}
			},
			expectedStatus: http.StatusCreated,
			expectedBody:   "new@example.com",
		},
		{
			name:           "ошибка: невалидный JSON",
			body:           "not a json",
			setupMock:      func(m *mockUserService) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "invalid request body",
		},
		{
			name: "ошибка: дубликат email",
			body: RegisterRequest{
				Email:    "dup@email.com",
				Username: "dupuser",
				Password: "securepass123",
			},
			setupMock: func(m *mockUserService) {
				m.registerFn = func(ctx context.Context, email, username, password string) (*domain.User, error) {
					return nil, userservice.ErrUserAlreadyExists
				}
			},
			expectedStatus: http.StatusConflict,
			expectedBody:   "already exists",
		},
		{
			name: "ошибка: внутренняя ошибка сервиса",
			body: RegisterRequest{
				Email:    "fail@example.com",
				Username: "failuser",
				Password: "securepass123",
			},
			setupMock: func(m *mockUserService) {
				m.registerFn = func(ctx context.Context, email, username, password string) (*domain.User, error) {
					return nil, errors.New("db connection failed")
				}
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   "internal server error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSvc := &mockUserService{}
			tt.setupMock(mockSvc)

			h := NewHandler(mockSvc)

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

			h.Register(rec, req)

			require.Equal(t, tt.expectedStatus, rec.Code)
			require.Contains(t, rec.Body.String(), tt.expectedBody)
		})
	}
}

func TestHandler_Login(t *testing.T) {
	tests := []struct {
		name           string
		body           interface{}
		setupMock      func(m *mockUserService)
		expectedStatus int
		expectedBody   string
	}{
		{
			name: "успешный логин",
			body: LoginRequest{
				Email:    "test@example.com",
				Password: "securepass123",
			},
			setupMock: func(m *mockUserService) {
				m.loginFn = func(ctx context.Context, email, password string) (string, *domain.User, error) {
					return "jwt-token-123", &domain.User{ID: 1, Email: email}, nil
				}
			},
			expectedStatus: http.StatusOK,
			expectedBody:   "jwt-token-123",
		},
		{
			name:           "ошибка: невалидный JSON",
			body:           "{broken",
			setupMock:      func(m *mockUserService) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "invalid request body",
		},
		{
			name: "ошибка: неверный credentials",
			body: LoginRequest{
				Email:    "test@example.com",
				Password: "wrongpass",
			},
			setupMock: func(m *mockUserService) {
				m.loginFn = func(ctx context.Context, email, password string) (string, *domain.User, error) {
					return "", nil, userservice.ErrInvalidCredentials
				}
			},
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   "invalid email or password",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSvc := &mockUserService{}
			tt.setupMock(mockSvc)

			h := NewHandler(mockSvc)

			var bodyBytes []byte
			switch v := tt.body.(type) {
			case string:
				bodyBytes = []byte(v)
			default:
				bodyBytes, _ = json.Marshal(v)
			}

			req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			h.Login(rec, req)

			require.Equal(t, tt.expectedStatus, rec.Code)
			require.Contains(t, rec.Body.String(), tt.expectedBody)
		})
	}
}
