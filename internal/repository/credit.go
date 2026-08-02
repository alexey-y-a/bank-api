package repository

import (
	"context"

	"github.com/alexey-y-a/bank-api/internal/domain"
)

type CreditRepository interface {
	CreateCredit(ctx context.Context, credit *domain.Credit) error
	FindByID(ctx context.Context, id int64) (*domain.Credit, error)
	FindByAccountID(ctx context.Context, accountID int64) ([]*domain.Credit, error)
	UpdateStatus(ctx context.Context, id int64, status domain.CreditStatus) error
	CreateScheduleItem(ctx context.Context, item *domain.CreditScheduleItem) error
	FindScheduleByCreditID(ctx context.Context, creditID int64) ([]*domain.CreditScheduleItem, error)
}
