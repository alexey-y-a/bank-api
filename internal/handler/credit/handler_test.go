package credit

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alexey-y-a/bank-api/internal/domain"
	"github.com/alexey-y-a/bank-api/internal/middleware"
	creditservice "github.com/alexey-y-a/bank-api/internal/service/credit"
	"github.com/stretchr/testify/require"
)

type mockService struct {
	createCreditFn func(ctx context.Context, accountID, userID int64, amount int64, rate float64, term int64) (*domain.Credit, []*domain.CreditScheduleItem, error)
	getScheduleFn  func(ctx context.Context, creditID, userID int64) ([]*domain.CreditScheduleItem, error)
	makePaymentFn  func(ctx context.Context, creditID, userID int64) error
}

func (m *mockService) CreateCredit(ctx context.Context, accountID, userID int64, amount int64, rate float64, term int64) (*domain.Credit, []*domain.CreditScheduleItem, error) {
	return m.createCreditFn(ctx, accountID, userID, amount, rate, term)
}

func (m *mockService) GetSchedule(ctx context.Context, creditID, userID int64) ([]*domain.CreditScheduleItem, error) {
	return m.getScheduleFn(ctx, creditID, userID)
}

func (m *mockService) MakePayment(ctx context.Context, creditID, userID int64) error {
	return m.makePaymentFn(ctx, creditID, userID)
}

func TestCreateCredit(t *testing.T) {
	tests := []struct {
		name           string
		userID         string
		body           string
		setupMock      func(m *mockService)
		expectedStatus int
	}{
		{
			name:   "успешное оформление кредита",
			userID: "5",
			body:   `{"account_id":1,"amount":10000000,"rate":15.0,"term":12}`,
			setupMock: func(m *mockService) {
				m.createCreditFn = func(ctx context.Context, accountID, userID, amount int64, rate float64, term int64) (*domain.Credit, []*domain.CreditScheduleItem, error) {
					schedule := make([]*domain.CreditScheduleItem, term)
					for i := range schedule {
						schedule[i] = &domain.CreditScheduleItem{ID: int64(i + 1), Status: domain.PaymentStatusPending}
					}

					return &domain.Credit{ID: 1, AccountID: accountID, Amount: amount, Rate: rate, TermMonths: term, Status: domain.CreditStatusActive}, schedule, nil
				}
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name:           "ошибка: нет авторизации",
			userID:         "",
			body:           `{"account_id":999,"amount":10000000,"rate":15.0,"term":12}`,
			setupMock:      func(m *mockService) {},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:   "ошибка: счет не найден",
			userID: "5",
			body:   `{"account_id":999,"amount":10000000,"rate":15.0,"term":12}`,
			setupMock: func(m *mockService) {
				m.createCreditFn = func(ctx context.Context, accountID, userID, amount int64, rate float64, term int64) (*domain.Credit, []*domain.CreditScheduleItem, error) {
					return nil, nil, creditservice.ErrCreditNotFound
				}
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:   "ошибка: не владелец счета",
			userID: "5",
			body:   `{"account_id":1,"amount":10000000,"rate":15.0,"term":12}`,
			setupMock: func(m *mockService) {
				m.createCreditFn = func(ctx context.Context, accountID, userID int64, amount int64, rate float64, term int64) (*domain.Credit, []*domain.CreditScheduleItem, error) {
					return nil, nil, creditservice.ErrForbidden
				}
			},
			expectedStatus: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockService{}
			tt.setupMock(mock)

			hdl := NewHandler(mock)

			req := httptest.NewRequest(http.MethodPost, "/credits", bytes.NewReader([]byte(tt.body)))
			req.Header.Set("Content-Type", "application/json")
			if tt.userID != "" {
				req = req.WithContext(middleware.WithUserID(req.Context(), tt.userID))
			}

			rec := httptest.NewRecorder()
			hdl.CreateCredit(rec, req)

			require.Equal(t, tt.expectedStatus, rec.Code)
		})
	}
}

func TestGetSchedule(t *testing.T) {
	tests := []struct {
		name           string
		userID         string
		creditID       string
		setupMock      func(m *mockService)
		expectedStatus int
		expectedLen    int
	}{
		{
			name:     "успешное получение графика",
			userID:   "5",
			creditID: "1",
			setupMock: func(m *mockService) {
				m.getScheduleFn = func(ctx context.Context, creditID, userID int64) ([]*domain.CreditScheduleItem, error) {
					return []*domain.CreditScheduleItem{
						{ID: 1, CreditID: 1},
						{ID: 2, CreditID: 1},
					}, nil
				}
			},
			expectedStatus: http.StatusOK,
			expectedLen:    2,
		},
		{
			name:           "ошибка: нет авторизации",
			userID:         "",
			creditID:       "1",
			setupMock:      func(m *mockService) {},
			expectedStatus: http.StatusUnauthorized,
			expectedLen:    0,
		},
		{
			name:     "ошибка: кредит не найден",
			userID:   "5",
			creditID: "999",
			setupMock: func(m *mockService) {
				m.getScheduleFn = func(ctx context.Context, creditID, userID int64) ([]*domain.CreditScheduleItem, error) {
					return nil, creditservice.ErrCreditNotFound
				}
			},
			expectedStatus: http.StatusNotFound,
			expectedLen:    0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockService{}
			tt.setupMock(mock)

			hdl := NewHandler(mock)
			req := httptest.NewRequest(http.MethodGet, "/credits/"+tt.creditID+"/schedule", nil)
			req.SetPathValue("id", tt.creditID)
			if tt.userID != "" {
				req = req.WithContext(middleware.WithUserID(req.Context(), tt.userID))
			}

			rec := httptest.NewRecorder()
			hdl.GetSchedule(rec, req)

			require.Equal(t, tt.expectedStatus, rec.Code)

			if tt.expectedStatus == http.StatusOK {
				var resp []ScheduleItemResponse
				err := json.Unmarshal(rec.Body.Bytes(), &resp)
				require.NoError(t, err)
				require.Len(t, resp, tt.expectedLen)
			}
		})
	}
}

func TestMakePayment(t *testing.T) {
	tests := []struct {
		name           string
		userID         string
		creditID       string
		setupMock      func(m *mockService)
		expectedStatus int
	}{
		{
			name:     "успешный платеж",
			userID:   "5",
			creditID: "1",
			setupMock: func(m *mockService) {
				m.makePaymentFn = func(ctx context.Context, creditID, userID int64) error {
					return nil
				}
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "ошибка: нет авторизации",
			userID:         "",
			creditID:       "1",
			setupMock:      func(m *mockService) {},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:     "ошибка: недостаточно средств",
			userID:   "5",
			creditID: "1",
			setupMock: func(m *mockService) {
				m.makePaymentFn = func(ctx context.Context, creditID, userID int64) error {
					return creditservice.ErrInsufficientFunds
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
			req := httptest.NewRequest(http.MethodPost, "/credits/"+tt.creditID+"/pay", nil)
			req.SetPathValue("id", tt.creditID)

			if tt.userID != "" {
				req = req.WithContext(middleware.WithUserID(req.Context(), tt.userID))
			}

			rec := httptest.NewRecorder()
			hdl.MakePayment(rec, req)

			require.Equal(t, tt.expectedStatus, rec.Code)
		})
	}
}
