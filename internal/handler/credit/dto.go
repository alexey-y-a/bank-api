package credit

import (
	"time"

	"github.com/alexey-y-a/bank-api/internal/domain"
)

type CreateCreditRequest struct {
	AccountID int64   `json:"account_id"`
	Amount    int64   `json:"amount"`
	Rate      float64 `json:"rate"`
	Term      int64   `json:"term"`
}

type ScheduleItemResponse struct {
	PaymentDate      string `json:"payment_date"`
	Principal        int64  `json:"principal"`
	Interest         int64  `json:"interest"`
	Total            int64  `json:"total"`
	RemainingBalance int64  `json:"remaining_balance"`
	Status           string `json:"status"`
}

type CreditResponse struct {
	ID         int64   `json:"id"`
	AccountID  int64   `json:"account_id"`
	Amount     int64   `json:"amount"`
	Rate       float64 `json:"rate"`
	TermMonths int64   `json:"term_months"`
	Status     string  `json:"status"`
}

func toCreditResponse(c *domain.Credit) CreditResponse {
	return CreditResponse{
		ID:         c.ID,
		AccountID:  c.AccountID,
		Amount:     c.Amount,
		Rate:       c.Rate,
		TermMonths: c.TermMonths,
		Status:     string(c.Status),
	}
}

func toScheduleItemResponse(item *domain.CreditScheduleItem) ScheduleItemResponse {
	return ScheduleItemResponse{
		PaymentDate:      item.PaymentDate.Format(time.RFC3339),
		Principal:        item.Principal,
		Interest:         item.Interest,
		Total:            item.Total,
		RemainingBalance: item.RemainingBalance,
		Status:           string(item.Status),
	}
}

func toScheduleListResponse(items []*domain.CreditScheduleItem) []ScheduleItemResponse {
	result := make([]ScheduleItemResponse, len(items))

	for i, item := range items {
		result[i] = toScheduleItemResponse(item)
	}

	return result
}
