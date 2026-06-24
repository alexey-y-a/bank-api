package repository

import (
	"context"

	"github.com/alexey-y-a/bank-api/internal/domain"
)

type AccountRepository interface {
	Create(ctx context.Context, account *domain.Account) error
	FindByID(ctx context.Context, id int64) (*domain.Account, error)
	FindByUserID(ctx context.Context, userID int64) ([]*domain.Account, error)
	UpdateBalance(ctx context.Context, id int64, balance int64) error
}
