package analytics

import (
	"context"
	"fmt"
	"time"

	"github.com/alexey-y-a/bank-api/internal/domain"
	"github.com/alexey-y-a/bank-api/internal/repository"
)

const topOperationsLimit = 5

type Service struct {
	analytics repository.AnalyticsReader
	accounts  repository.AccountRepository
}

func NewService(analytics repository.AnalyticsReader, accounts repository.AccountRepository) *Service {
	return &Service{
		analytics: analytics,
		accounts:  accounts,
	}
}

func (s *Service) GetIncomeExpense(ctx context.Context, userID int64, from, to time.Time) (domain.PeriodSummary, []*domain.Event, error) {
	accounts, err := s.accounts.FindByUserID(ctx, userID)
	if err != nil {
		return domain.PeriodSummary{}, nil, fmt.Errorf("find user accounts: %w", err)
	}

	if len(accounts) == 0 {
		return domain.PeriodSummary{}, nil, nil
	}

	accountIDs := make([]int64, 0, len(accounts))

	for _, account := range accounts {
		accountIDs = append(accountIDs, account.ID)
	}

	summary, err := s.analytics.IncomeExpense(ctx, accountIDs, from, to)
	if err != nil {
		return domain.PeriodSummary{}, nil, fmt.Errorf("get income/expense: %w", err)
	}

	topOperations, err := s.analytics.TopOperations(ctx, accountIDs, from, to, topOperationsLimit)
	if err != nil {
		return domain.PeriodSummary{}, nil, fmt.Errorf("get top operations: %w", err)
	}

	return summary, topOperations, nil
}
