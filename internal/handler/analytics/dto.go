package analytics

import (
	"time"

	"github.com/alexey-y-a/bank-api/internal/domain"
)

type IncomeExpenseResponse struct {
	Month         string              `json:"month"`
	Income        int64               `json:"income"`
	Expense       int64               `json:"expense"`
	TopOperations []OperationResponse `json:"top_operations"`
}

type OperationResponse struct {
	EventID       string `json:"event_id"`
	Type          string `json:"type"`
	FromAccountID int64  `json:"from_account_id"`
	ToAccountID   int64  `json:"to_account_id"`
	Amount        int64  `json:"amount"`
	CreatedAt     string `json:"created_at"`
}

func toOperationResponse(event *domain.Event) OperationResponse {
	return OperationResponse{
		EventID:       event.ID,
		Type:          string(event.Type),
		FromAccountID: event.FromAccountID,
		ToAccountID:   event.ToAccountID,
		Amount:        event.Amount,
		CreatedAt:     event.CreatedAt.UTC().Format(time.RFC3339),
	}
}

func toIncomeExpenseResponse(month string, summary domain.PeriodSummary, events []*domain.Event) IncomeExpenseResponse {
	operations := make([]OperationResponse, 0, len(events))

	for _, event := range events {
		operations = append(operations, toOperationResponse(event))
	}

	return IncomeExpenseResponse{
		Month:         month,
		Income:        summary.Income,
		Expense:       summary.Expense,
		TopOperations: operations,
	}
}
