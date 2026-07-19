package account

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alexey-y-a/bank-api/internal/domain"
	"github.com/alexey-y-a/bank-api/internal/middleware"
	accountservice "github.com/alexey-y-a/bank-api/internal/service/account"
	"github.com/stretchr/testify/require"
)

type mockService struct {
	createFn      func(ctx context.Context, userID int64, currency string) (*domain.Account, error)
	getByIDFn     func(ctx context.Context, accountID, userID int64) (*domain.Account, error)
	getByUserIDFn func(ctx context.Context, userID int64) ([]*domain.Account, error)
	depositFn     func(ctx context.Context, accountID, userID, amount int64) (*domain.Account, error)
	withdrawFn    func(ctx context.Context, accountID, userID, amount int64) (*domain.Account, error)
}

func (m *mockService) Create(ctx context.Context, userID int64, currency string) (*domain.Account, error) {
	return m.createFn(ctx, userID, currency)
}

func (m *mockService) GetByID(ctx context.Context, accountID, userID int64) (*domain.Account, error) {
	return m.getByIDFn(ctx, accountID, userID)
}

func (m *mockService) GetByUserID(ctx context.Context, userID int64) ([]*domain.Account, error) {
	return m.getByUserIDFn(ctx, userID)
}

func (m *mockService) Deposit(ctx context.Context, accountID, userID, amount int64) (*domain.Account, error) {
	return m.depositFn(ctx, accountID, userID, amount)
}

func (m *mockService) Withdraw(ctx context.Context, accountID, userID, amount int64) (*domain.Account, error) {
	return m.withdrawFn(ctx, accountID, userID, amount)
}

func TestHandler_CreateAccount(t *testing.T) {
	t.Run("успешное создание счета", func(t *testing.T) {
		mock := &mockService{
			createFn: func(ctx context.Context, userID int64, currency string) (*domain.Account, error) {
				return &domain.Account{ID: 1, UserID: userID, Balance: 0, Currency: currency}, nil
			},
		}

		hdl := NewHandler(mock)

		body := `{"currency":"RUB"}`
		req := httptest.NewRequest(http.MethodPost, "/accounts", bytes.NewReader([]byte(body)))
		req.Header.Set("Content-Type", "application/json")
		req = req.WithContext(middleware.WithUserID(req.Context(), "5"))

		rec := httptest.NewRecorder()
		hdl.CreateAccount(rec, req)

		require.Equal(t, http.StatusCreated, rec.Code, "должен быть 201 Created")

		var resp AccountResponse
		err := json.Unmarshal(rec.Body.Bytes(), &resp)
		require.NoError(t, err, "ответ должен быть валидным JSON")
		require.Equal(t, int64(1), resp.ID, "ID счета должен быть 1")
		require.Equal(t, "RUB", resp.Currency, "валюта должна быть RUB")
	})

	t.Run("ошибка: нет авторизации", func(t *testing.T) {
		mock := &mockService{}
		hdl := NewHandler(mock)

		body := `{"currency":"RUB"}`
		req := httptest.NewRequest(http.MethodPost, "/accounts", bytes.NewReader([]byte(body)))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		hdl.CreateAccount(rec, req)

		require.Equal(t, http.StatusUnauthorized, rec.Code, "должен быть 401 Unauthorized")
	})

	t.Run("ошибка: невалидный JSON", func(t *testing.T) {
		mock := &mockService{}
		hdl := NewHandler(mock)

		req := httptest.NewRequest(http.MethodPost, "/accounts", bytes.NewReader([]byte("not json")))
		req.Header.Set("Content-Type", "application/json")
		req = req.WithContext(middleware.WithUserID(req.Context(), "5"))
		rec := httptest.NewRecorder()
		hdl.CreateAccount(rec, req)

		require.Equal(t, http.StatusBadRequest, rec.Code, "должен быть 400 BadRequest")
	})

	t.Run("ошибка: сбой сервиса", func(t *testing.T) {
		mock := &mockService{
			createFn: func(ctx context.Context, userID int64, currency string) (*domain.Account, error) {
				return nil, errors.New("db connection failed")
			},
		}
		hdl := NewHandler(mock)
		body := `{"currency":"RUB"}`
		req := httptest.NewRequest(http.MethodPost, "/accounts", bytes.NewReader([]byte(body)))
		req.Header.Set("Content-Type", "application/json")
		req = req.WithContext(middleware.WithUserID(req.Context(), "5"))
		rec := httptest.NewRecorder()
		hdl.CreateAccount(rec, req)

		require.Equal(t, http.StatusInternalServerError, rec.Code, "должен быть 500")
	})
}

func TestHandler_GetAccount(t *testing.T) {
	t.Run("успешное получение счета", func(t *testing.T) {
		mock := &mockService{
			getByIDFn: func(ctx context.Context, accountID, userID int64) (*domain.Account, error) {
				return &domain.Account{ID: accountID, UserID: userID, Balance: 5000}, nil
			},
		}
		hdl := NewHandler(mock)

		req := httptest.NewRequest(http.MethodGet, "/accounts/1", nil)
		req.SetPathValue("id", "1")
		req = req.WithContext(middleware.WithUserID(req.Context(), "5"))
		rec := httptest.NewRecorder()
		hdl.GetAccount(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)

		var resp AccountResponse
		err := json.Unmarshal(rec.Body.Bytes(), &resp)
		require.NoError(t, err)
		require.Equal(t, int64(1), resp.ID)
		require.Equal(t, int64(5000), resp.Balance)
	})

	t.Run("ошибка: счет не найден", func(t *testing.T) {
		mock := &mockService{
			getByIDFn: func(ctx context.Context, accountID, userID int64) (*domain.Account, error) {
				return nil, accountservice.ErrAccountNotFound
			},
		}
		hdl := NewHandler(mock)

		req := httptest.NewRequest(http.MethodGet, "/accounts/999", nil)
		req.SetPathValue("id", "999")
		req = req.WithContext(middleware.WithUserID(req.Context(), "5"))
		rec := httptest.NewRecorder()
		hdl.GetAccount(rec, req)

		require.Equal(t, http.StatusNotFound, rec.Code, "должен быть 404")
	})

	t.Run("ошибка доступ к чужому счету", func(t *testing.T) {
		mock := &mockService{
			getByIDFn: func(ctx context.Context, accountID, userID int64) (*domain.Account, error) {
				return nil, accountservice.ErrForbidden
			},
		}

		hdl := NewHandler(mock)

		req := httptest.NewRequest(http.MethodGet, "/accounts/1", nil)
		req.SetPathValue("id", "1")
		req = req.WithContext(middleware.WithUserID(req.Context(), "2"))
		rec := httptest.NewRecorder()
		hdl.GetAccount(rec, req)

		require.Equal(t, http.StatusForbidden, rec.Code, "должен быть 403")
	})
}

func TestHandler_GetUserAccounts(t *testing.T) {
	t.Run("успешное получение списка счетов", func(t *testing.T) {
		mock := &mockService{
			getByUserIDFn: func(ctx context.Context, userID int64) ([]*domain.Account, error) {
				return []*domain.Account{
					{ID: 1, UserID: userID, Balance: 1000},
					{ID: 2, UserID: userID, Balance: 2000},
				}, nil
			},
		}

		hdl := NewHandler(mock)

		req := httptest.NewRequest(http.MethodGet, "/accounts", nil)
		req = req.WithContext(middleware.WithUserID(req.Context(), "5"))
		rec := httptest.NewRecorder()
		hdl.GetUserAccounts(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)

		var resp []AccountResponse
		err := json.Unmarshal(rec.Body.Bytes(), &resp)
		require.NoError(t, err)
		require.Len(t, resp, 2, "должно быть 2 счета")
	})
}

func TestHandler_Deposit(t *testing.T) {
	t.Run("успешное пополнение", func(t *testing.T) {
		mock := &mockService{
			depositFn: func(ctx context.Context, accountID, userID int64, amount int64) (*domain.Account, error) {
				return &domain.Account{ID: accountID, UserID: userID, Balance: amount}, nil
			},
		}

		hdl := NewHandler(mock)

		body := `{"amount":5000}`
		req := httptest.NewRequest(http.MethodPost, "/accounts/1/deposit", bytes.NewReader([]byte(body)))
		req.Header.Set("Content-Type", "application/json")
		req.SetPathValue("id", "1")
		req = req.WithContext(middleware.WithUserID(req.Context(), "5"))
		rec := httptest.NewRecorder()
		hdl.Deposit(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)

		var resp AccountResponse
		err := json.Unmarshal(rec.Body.Bytes(), &resp)
		require.NoError(t, err)
		require.Equal(t, int64(5000), resp.Balance)
	})

	t.Run("ошибка невалидная сумма", func(t *testing.T) {
		mock := &mockService{
			depositFn: func(ctx context.Context, accountID, userID int64, amount int64) (*domain.Account, error) {
				return nil, domain.ErrInvalidAmount
			},
		}

		hdl := NewHandler(mock)

		body := `{"amount":0}`
		req := httptest.NewRequest(http.MethodPost, "/accounts/1/deposit", bytes.NewReader([]byte(body)))
		req.Header.Set("Content-Type", "application/json")
		req.SetPathValue("id", "1")
		req = req.WithContext(middleware.WithUserID(req.Context(), "5"))
		rec := httptest.NewRecorder()
		hdl.Deposit(rec, req)

		require.Equal(t, http.StatusBadRequest, rec.Code, "должно быть 400")
	})
}

func TestHandler_Withdraw(t *testing.T) {
	t.Run("успешное списание", func(t *testing.T) {
		mock := &mockService{
			withdrawFn: func(ctx context.Context, accountID, userID int64, amount int64) (*domain.Account, error) {
				return &domain.Account{ID: accountID, UserID: userID, Balance: 3000}, nil
			},
		}

		hdl := NewHandler(mock)

		body := `{"amount":2000}`
		req := httptest.NewRequest(http.MethodPost, "/accounts/1/withdraw", bytes.NewReader([]byte(body)))
		req.Header.Set("Content-Type", "application/json")
		req.SetPathValue("id", "1")
		req = req.WithContext(middleware.WithUserID(req.Context(), "5"))
		rec := httptest.NewRecorder()
		hdl.Withdraw(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)

		var resp AccountResponse
		err := json.Unmarshal(rec.Body.Bytes(), &resp)
		require.NoError(t, err)
		require.Equal(t, int64(3000), resp.Balance)
	})

	t.Run("ошибка: недостаточно средств", func(t *testing.T) {
		mock := &mockService{
			withdrawFn: func(ctx context.Context, accountID, userID int64, amount int64) (*domain.Account, error) {
				return nil, domain.ErrInsufficientFunds
			},
		}

		hdl := NewHandler(mock)
		body := `{"amount":99999}`
		req := httptest.NewRequest(http.MethodPost, "/accounts/1/withdraw", bytes.NewReader([]byte(body)))
		req.Header.Set("Content-Type", "application/json")
		req.SetPathValue("id", "1")
		req = req.WithContext(middleware.WithUserID(req.Context(), "5"))
		rec := httptest.NewRecorder()
		hdl.Withdraw(rec, req)

		require.Equal(t, http.StatusBadRequest, rec.Code, "должен быть 400")
	})
}
