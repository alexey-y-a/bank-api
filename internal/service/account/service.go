package account

import (
	"context"
	"fmt"

	"github.com/alexey-y-a/bank-api/internal/domain"
	"github.com/alexey-y-a/bank-api/internal/repository"
)

type Service struct {
	repo repository.AccountRepository
}

func NewService(repo repository.AccountRepository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Create(ctx context.Context, userID int64, currency string) (*domain.Account, error) {
	account := domain.NewAccount(userID, currency)

	err := s.repo.Create(ctx, account)
	if err != nil {
		return nil, fmt.Errorf("account_service.Create: %w", err)
	}

	return account, nil
}

func (s *Service) GetByID(ctx context.Context, accountID, userID int64) (*domain.Account, error) {
	account, err := s.repo.FindByID(ctx, accountID)
	if err != nil {
		return nil, fmt.Errorf("account_service.FindByID: %w", err)
	}

	if account == nil {
		return nil, ErrAccountNotFound
	}

	if account.UserID != userID {
		return nil, ErrForbidden
	}

	return account, nil
}

func (s *Service) GetByUserID(ctx context.Context, userID int64) ([]*domain.Account, error) {
	accounts, err := s.repo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("account_service.FindByUserID: %w", err)
	}

	return accounts, nil
}

func (s *Service) Deposit(ctx context.Context, accountID, userID, amount int64) (*domain.Account, error) {
	account, err := s.GetByID(ctx, accountID, userID)
	if err != nil {
		return nil, err
	}

	err = account.Deposit(amount)
	if err != nil {
		return nil, domain.ErrInvalidAmount
	}

	err = s.repo.UpdateBalance(ctx, account.ID, account.Balance)
	if err != nil {
		return nil, fmt.Errorf("account_service.UpdateBalance: %w", err)
	}

	return account, nil
}

func (s *Service) Withdraw(ctx context.Context, accountID, userID, amount int64) (*domain.Account, error) {
	account, err := s.GetByID(ctx, accountID, userID)
	if err != nil {
		return nil, err
	}

	err = account.Withdraw(amount)
	if err != nil {
		return nil, domain.ErrInsufficientFunds
	}

	err = s.repo.UpdateBalance(ctx, account.ID, account.Balance)
	if err != nil {
		return nil, fmt.Errorf("account_service.UpdateBalance: %w", err)
	}

	return account, nil
}
