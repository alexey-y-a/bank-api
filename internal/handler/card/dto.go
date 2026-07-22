package card

import (
	"time"

	"github.com/alexey-y-a/bank-api/internal/domain"
)

type CreateCardRequest struct {
	AccountID int64 `json:"account_id"`
}

type PayWithCardRequest struct {
	CVV    string `json:"cvv"`
	Amount int64  `json:"amount"`
}

type CardResponse struct {
	ID        int64  `json:"id"`
	AccountID int64  `json:"account_id"`
	Number    string `json:"number"`
	ExpiresAt string `json:"expires_at"`
	Status    string `json:"status"`
	CreatedAt string `json:"created_at"`
}

func toCardResponse(c *domain.Card) CardResponse {
	return CardResponse{
		ID:        c.ID,
		AccountID: c.AccountID,
		Number:    c.MaskNumber(),
		ExpiresAt: c.ExpiresAt.Format(time.RFC3339),
		Status:    string(c.Status),
		CreatedAt: c.CreatedAt.Format(time.RFC3339),
	}
}

func toCardListResponse(cards []*domain.Card) []CardResponse {
	result := make([]CardResponse, len(cards))

	for i, c := range cards {
		result[i] = toCardResponse(c)
	}

	return result
}
