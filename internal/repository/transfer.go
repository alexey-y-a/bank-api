package repository

import (
	"context"

	"github.com/alexey-y-a/bank-api/internal/domain"
)

type TransferRepository interface {
	Create(ctx context.Context, tx *domain.Transaction) error
	GetByAccountID(ctx context.Context, accountID int64, limit, offset int) ([]*domain.Transaction, error)
	GetByUserID(ctx context.Context, userID int64, limit, offset int) ([]*domain.Transaction, error)
}
