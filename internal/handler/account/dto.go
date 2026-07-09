package account

import (
	"time"

	"github.com/alexey-y-a/bank-api/internal/domain"
)

type CreateAccountRequest struct {
	Currency string `json:"currency"`
}

type DepositRequest struct {
	Amount int64 `json:"amount"`
}

type WithdrawRequest struct {
	Amount int64 `json:"amount"`
}

type AccountResponse struct {
	ID        int64  `json:"id"`
	UserID    int64  `json:"user_id"`
	Balance   int64  `json:"balance"`
	Currency  string `json:"currency"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

func toAccountResponse(a *domain.Account) AccountResponse {
	return AccountResponse{
		ID:        a.ID,
		UserID:    a.UserID,
		Balance:   a.Balance,
		Currency:  a.Currency,
		CreatedAt: a.CreatedAt.Format(time.RFC3339),
		UpdatedAt: a.UpdatedAt.Format(time.RFC3339),
	}
}

func toAccountListResponse(accounts []*domain.Account) []AccountResponse {
	result := make([]AccountResponse, len(accounts))

	for i, a := range accounts {
		result[i] = toAccountResponse(a)
	}

	return result
}
