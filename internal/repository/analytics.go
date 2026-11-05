package repository

import (
	"context"

	"github.com/alexey-y-a/bank-api/internal/domain"
)

type AnalyticsRepository interface {
	SaveBatch(ctx context.Context, event []*domain.Event) error
}
