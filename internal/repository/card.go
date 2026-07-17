package repository

import (
	"context"

	"github.com/alexey-y-a/bank-api/internal/domain"
)

type CardRepository interface {
	Create(ctx context.Context, card *domain.Card) error
	FindByID(ctx context.Context, id int64) (*domain.Card, error)
	FindByAccountID(ctx context.Context, accountID int64) ([]*domain.Card, error)
	UpdateStatus(ctx context.Context, id int64, status domain.CardStatus) error
}
