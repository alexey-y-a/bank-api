package repository

import (
	"context"
	"time"

	"github.com/alexey-y-a/bank-api/internal/domain"
)

type AnalyticsWriter interface {
	SaveBatch(ctx context.Context, event []*domain.Event) error
}

type AnalyticsReader interface {
	IncomeExpense(ctx context.Context, accountIDs []int64, from, to time.Time) (domain.PeriodSummary, error)
	TopOperations(ctx context.Context, accountIDs []int64, from, to time.Time, limit int) ([]*domain.Event, error)
}
