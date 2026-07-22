package card

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alexey-y-a/bank-api/internal/domain"
	"github.com/alexey-y-a/bank-api/internal/middleware"
	cardservice "github.com/alexey-y-a/bank-api/internal/service/card"
	"github.com/stretchr/testify/require"
)

type mockService struct {
	issueCardFn    func(ctx context.Context, accountId, userID int64) (*domain.Card, error)
	getUserCardsFn func(ctx context.Context, userID int64) ([]*domain.Card, error)
	blockCardFn    func(ctx context.Context, cardID, userID int64) error
	payWithCardFn  func(ctx context.Context, cardID, userID int64, cvv string, amount int64) error
}

func (m *mockService) IssueCard(ctx context.Context, accountId, userID int64) (*domain.Card, error) {
	return m.issueCardFn(ctx, accountId, userID)
}

func (m *mockService) GetUserCards(ctx context.Context, userID int64) ([]*domain.Card, error) {
	return m.getUserCardsFn(ctx, userID)
}

func (m *mockService) BlockCard(ctx context.Context, cardID, userID int64) error {
	return m.blockCardFn(ctx, cardID, userID)
}

func (m *mockService) PayWithCard(ctx context.Context, cardID, userID int64, cvv string, amount int64) error {
	return m.payWithCardFn(ctx, cardID, userID, cvv, amount)
}

func TestHandler_CreateCard(t *testing.T) {
	tests := []struct {
		name           string
		userID         string
		body           string
		setupMock      func(m *mockService)
		expectedStatus int
		checkResponse  func(t *testing.T, rec *httptest.ResponseRecorder)
	}{
		{
			name:   "успешный выпуск карты",
			userID: "5",
			body:   `{"account_id": 1}`,
			setupMock: func(m *mockService) {
				m.issueCardFn = func(ctx context.Context, accountId, userID int64) (*domain.Card, error) {
					return &domain.Card{
						ID:        1,
						AccountID: accountId,
						Number:    "4532148813416220",
						Status:    domain.CardStatusActive,
					}, nil
				}
			},
			expectedStatus: http.StatusCreated,
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var resp CardResponse
				err := json.Unmarshal(rec.Body.Bytes(), &resp)
				require.NoError(t, err, "ответ должен быть валидным")
				require.Equal(t, int64(1), resp.ID, "ID карты должен быть 1")
				require.Equal(t, "**** **** **** 6220", resp.Number, "номер должен быть маскирован")
			},
		},
		{
			name:   "ошибка: нет авторизации",
			userID: "",
			body:   `{"account_id": 1}`,
			setupMock: func(m *mockService) {
				m.issueCardFn = func(ctx context.Context, accountId, userID int64) (*domain.Card, error) {
					return nil, cardservice.ErrCardNotFound
				}
			},
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockService{}
			tt.setupMock(mock)

			hdl := NewHandler(mock)

			req := httptest.NewRequest(http.MethodPost, "/cards", bytes.NewReader([]byte(tt.body)))
			req.Header.Set("Content-Type", "application/json")

			if tt.userID != "" {
				req = req.WithContext(middleware.WithUserID(req.Context(), tt.userID))
			}

			rec := httptest.NewRecorder()
			hdl.CreateCard(rec, req)

			require.Equal(t, tt.expectedStatus, rec.Code, "HTTP-статус должен совпадать")

			if tt.checkResponse != nil {
				tt.checkResponse(t, rec)
			}
		})
	}
}

func TestHandler_GetUserCards(t *testing.T) {
	tests := []struct {
		name           string
		userID         string
		setupMock      func(m *mockService)
		expectedStatus int
		checkResponse  func(t *testing.T, rec *httptest.ResponseRecorder)
	}{
		{
			name:   "успешное получение списка",
			userID: "5",
			setupMock: func(m *mockService) {
				m.getUserCardsFn = func(ctx context.Context, userID int64) ([]*domain.Card, error) {
					return []*domain.Card{
						{ID: 1, AccountID: 1, Number: "4532148813416220"},
						{ID: 2, AccountID: 1, Number: "5555555555554444"},
					}, nil
				}
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var resp []CardResponse
				err := json.Unmarshal(rec.Body.Bytes(), &resp)
				require.NoError(t, err, "ответ должен быть валидным JSON")
				require.Len(t, resp, 2, "должно быть 2 карты")
			},
		},
		{
			name:           "ошибка нет авторизации",
			userID:         "",
			setupMock:      func(m *mockService) {},
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockService{}
			tt.setupMock(mock)

			hdl := NewHandler(mock)

			req := httptest.NewRequest(http.MethodGet, "/cards", nil)
			if tt.userID != "" {
				req = req.WithContext(middleware.WithUserID(req.Context(), tt.userID))
			}

			rec := httptest.NewRecorder()
			hdl.GetUserCards(rec, req)

			require.Equal(t, tt.expectedStatus, rec.Code)

			if tt.checkResponse != nil {
				tt.checkResponse(t, rec)
			}
		})
	}
}

func TestHandler_BlockCard(t *testing.T) {
	tests := []struct {
		name           string
		userID         string
		cardID         string
		setupMock      func(m *mockService)
		expectedStatus int
	}{
		{
			name:   "успешная блокировка",
			userID: "5",
			cardID: "1",
			setupMock: func(m *mockService) {
				m.blockCardFn = func(ctx context.Context, cardID, userID int64) error {
					return nil
				}
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:   "ошибка карта не найдена",
			userID: "5",
			cardID: "999",
			setupMock: func(m *mockService) {
				m.blockCardFn = func(ctx context.Context, cardID, userID int64) error {
					return cardservice.ErrCardNotFound
				}
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "ошибка нет авторизации",
			userID:         "",
			cardID:         "1",
			setupMock:      func(m *mockService) {},
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockService{}
			tt.setupMock(mock)

			hdl := NewHandler(mock)

			req := httptest.NewRequest(http.MethodPost, "/cards/"+tt.cardID+"/block", nil)
			req.SetPathValue("id", tt.cardID)
			if tt.userID != "" {
				req = req.WithContext(middleware.WithUserID(req.Context(), tt.userID))
			}

			rec := httptest.NewRecorder()
			hdl.BlockCard(rec, req)

			require.Equal(t, tt.expectedStatus, rec.Code)
		})
	}
}

func TestHandler_PayWithCard(t *testing.T) {
	tests := []struct {
		name           string
		userID         string
		cardID         string
		body           string
		setupMock      func(m *mockService)
		expectedStatus int
	}{
		{
			name:   "успешная оплата",
			userID: "5",
			cardID: "1",
			body:   `{"cvv":"123","amount":5000}`,
			setupMock: func(m *mockService) {
				m.payWithCardFn = func(ctx context.Context, cardID, userID int64, cvv string, amount int64) error {
					return nil
				}
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:   "ошибка: неверный CVV",
			userID: "5",
			cardID: "1",
			body:   `{"cvv":"999","amount":5000}`,
			setupMock: func(m *mockService) {
				m.payWithCardFn = func(ctx context.Context, cardID, userID int64, cvv string, amount int64) error {
					return cardservice.ErrInvalidCVV
				}
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "ошибка: нет авторизации",
			userID:         "",
			cardID:         "1",
			body:           `{"cvv":"123","amount":5000}`,
			setupMock:      func(m *mockService) {},
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockService{}
			tt.setupMock(mock)

			hdl := NewHandler(mock)

			req := httptest.NewRequest(http.MethodPost, "/cards/"+tt.cardID+"/pay", bytes.NewReader([]byte(tt.body)))
			req.Header.Set("Content-Type", "application/json")
			req.SetPathValue("id", tt.cardID)
			if tt.userID != "" {
				req = req.WithContext(middleware.WithUserID(req.Context(), tt.userID))
			}

			rec := httptest.NewRecorder()
			hdl.PayWithCard(rec, req)

			require.Equal(t, tt.expectedStatus, rec.Code)
		})
	}
}
