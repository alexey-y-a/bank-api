package transfer

import (
	"time"

	"github.com/alexey-y-a/bank-api/internal/domain"
)

type TransferRequest struct {
	FromAccountID int64 `json:"from_account_id"`
	ToAccountID   int64 `json:"to_account_id"`
	Amount        int64 `json:"amount"`
}

type HistoryResponse struct {
	ID            int64  `json:"id"`
	FromAccountID int64  `json:"from_account_id"`
	ToAccountID   int64  `json:"to_account_id"`
	Amount        int64  `json:"amount"`
	Type          string `json:"type"`
	Description   string `json:"description"`
	CreatedAt     string `json:"created_at"`
}

func toHistoryResponse(t *domain.Transaction) HistoryResponse {
	return HistoryResponse{
		ID:            t.ID,
		FromAccountID: t.FromAccountID,
		ToAccountID:   t.ToAccountID,
		Amount:        t.Amount,
		Type:          string(t.Type),
		Description:   t.Description,
		CreatedAt:     t.CreatedAt.Format(time.RFC3339),
	}
}

func toHistoryListResponse(transactions []*domain.Transaction) []HistoryResponse {
	result := make([]HistoryResponse, len(transactions))

	for i, t := range transactions {
		result[i] = toHistoryResponse(t)
	}

	return result
}
