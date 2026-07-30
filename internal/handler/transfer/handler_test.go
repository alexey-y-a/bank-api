package transfer

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alexey-y-a/bank-api/internal/domain"
	"github.com/alexey-y-a/bank-api/internal/middleware"
	transferservice "github.com/alexey-y-a/bank-api/internal/service/transfer"
	"github.com/stretchr/testify/require"
)

type mockService struct {
	depositFn      func(ctx context.Context, accountID, userID, amount int64) error
	makeTransferFn func(ctx context.Context, fromAccountID, toAccountID, userID, amount int64) error
	getHistoryFn   func(ctx context.Context, userID int64, limit, offset int) ([]*domain.Transaction, error)
}

func (s *mockService) Deposit(ctx context.Context, accountID, userID, amount int64) error {
	return s.depositFn(ctx, accountID, userID, amount)
}

func (s *mockService) MakeTransfer(ctx context.Context, fromAccountID, toAccountID, userID, amount int64) error {
	return s.makeTransferFn(ctx, fromAccountID, toAccountID, userID, amount)
}

func (s *mockService) GetHistory(ctx context.Context, userID int64, limit, offset int) ([]*domain.Transaction, error) {
	return s.getHistoryFn(ctx, userID, limit, offset)
}

func TestMakeTransfer(t *testing.T) {
	tests := []struct {
		name           string
		userID         string
		body           string
		setupMock      func(m *mockService)
		expectedStatus int
	}{
		{
			name:   "успешный перевод",
			userID: "5",
			body:   `{"from_account_id":1,"to_account_id":2,"amount":5000}`,
			setupMock: func(m *mockService) {
				m.makeTransferFn = func(ctx context.Context, fromAccountID, toAccountID, userID, amount int64) error {
					return nil
				}
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "ошибка нет авторизации",
			userID:         "",
			body:           `{"from_account_id":1,"to_account_id":2,"amount":5000}`,
			setupMock:      func(m *mockService) {},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:   "ошибка счет не найден",
			userID: "5",
			body:   `{"from_account_id":999,"to_account_id":2,"amount":5000}`,
			setupMock: func(m *mockService) {
				m.makeTransferFn = func(ctx context.Context, fromAccountID, toAccountID, userID, amount int64) error {
					return transferservice.ErrAccountNotFound
				}
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:   "ошибка: недостаточно средств",
			userID: "5",
			body:   `{"from_account_id":1,"to_account_id":2,"amount":999999}`,
			setupMock: func(m *mockService) {
				m.makeTransferFn = func(ctx context.Context, fromAccountID, toAccountID, userID, amount int64) error {
					return transferservice.ErrInsufficientFunds
				}
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:   "ошибка: перевод на тот же счет",
			userID: "5",
			body:   `{"from_account_id":1,"to_account_id":1,"amount":5000}`,
			setupMock: func(m *mockService) {
				m.makeTransferFn = func(ctx context.Context, fromAccountID, toAccountID, userID, amount int64) error {
					return transferservice.ErrSameAccount
				}
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockService{}
			tt.setupMock(mock)

			hdl := NewHandler(mock)

			req := httptest.NewRequest(http.MethodPost, "/transfer", bytes.NewReader([]byte(tt.body)))
			req.Header.Set("Content-Type", "application/json")
			if tt.userID != "" {
				req = req.WithContext(middleware.WithUserID(req.Context(), tt.userID))
			}

			rec := httptest.NewRecorder()
			hdl.MakeTransfer(rec, req)

			require.Equal(t, tt.expectedStatus, rec.Code)
		})
	}
}

func TestGetHistory(t *testing.T) {
	tests := []struct {
		name           string
		userID         string
		setupMock      func(m *mockService)
		expectedStatus int
		expectedLen    int
	}{
		{
			name:   "успешное получение истории",
			userID: "5",
			setupMock: func(m *mockService) {
				m.getHistoryFn = func(ctx context.Context, userID int64, limit, offset int) ([]*domain.Transaction, error) {
					return []*domain.Transaction{
						{ID: 1, Amount: 5000},
						{ID: 2, Amount: 3000},
					}, nil
				}
			},
			expectedStatus: http.StatusOK,
			expectedLen:    2,
		},
		{
			name:           "ошибка: нет авторизации",
			userID:         "",
			setupMock:      func(m *mockService) {},
			expectedStatus: http.StatusUnauthorized,
			expectedLen:    0,
		},
		{
			name:   "пустая история",
			userID: "5",
			setupMock: func(m *mockService) {
				m.getHistoryFn = func(ctx context.Context, userID int64, limit, offset int) ([]*domain.Transaction, error) {
					return []*domain.Transaction{}, nil
				}
			},
			expectedStatus: http.StatusOK,
			expectedLen:    0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockService{}
			tt.setupMock(mock)

			hdl := NewHandler(mock)

			req := httptest.NewRequest(http.MethodGet, "/history", nil)
			if tt.userID != "" {
				req = req.WithContext(middleware.WithUserID(req.Context(), tt.userID))
			}

			rec := httptest.NewRecorder()
			hdl.GetHistory(rec, req)

			if tt.expectedStatus == http.StatusOK {
				var resp []HistoryResponse
				err := json.Unmarshal(rec.Body.Bytes(), &resp)
				require.NoError(t, err, "ответ должен быть валидным JSON")
				require.Len(t, resp, tt.expectedLen)
			}
		})
	}
}
