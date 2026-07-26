package transfer

import (
	"context"
	"fmt"

	"github.com/alexey-y-a/bank-api/internal/domain"
	"github.com/alexey-y-a/bank-api/internal/repository"
	"github.com/alexey-y-a/bank-api/internal/repository/postgres"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	debitAccountQuery  = "UPDATE accounts SET balance = $1, updated_at = NOW() WHERE id = $2"
	creditAccountQuery = "UPDATE accounts SET balance = $1, updated_at = NOW() WHERE id = $2"
)

type Service struct {
	accountRepo  repository.AccountRepository
	transferRepo repository.TransferRepository
	pool         *pgxpool.Pool
}

func NewService(accountRepo repository.AccountRepository, transferRepo repository.TransferRepository, pool *pgxpool.Pool) *Service {
	return &Service{
		accountRepo:  accountRepo,
		transferRepo: transferRepo,
		pool:         pool,
	}
}

func (s *Service) Deposit(ctx context.Context, accountID, userID int64, amount int64) error {
	account, err := s.accountRepo.FindByID(ctx, accountID)
	if err != nil {
		return fmt.Errorf("transfer_service.Deposit: %w", err)
	}

	if account == nil {
		return ErrAccountNotFound
	}

	if account.UserID != userID {
		return ErrForbidden
	}

	err = account.Deposit(amount)
	if err != nil {
		return ErrInsufficientFunds
	}

	err = s.accountRepo.UpdateBalance(ctx, accountID, account.Balance)
	if err != nil {
		return fmt.Errorf("transfer_service.Deposit update balance: %w", err)
	}

	tx, err := domain.NewTransaction(0, accountID, amount, domain.TransactionTypeDeposit, "deposit")
	if err != nil {
		return fmt.Errorf("transfer_service.Deposit new transaction: %w", err)
	}

	err = s.transferRepo.Create(ctx, tx)
	if err != nil {
		return fmt.Errorf("transfer_service.Deposit create (save) transaction: %w", err)
	}

	return nil
}

func (s *Service) MakeTransfer(ctx context.Context, fromAccountID, toAccountID, userId, amount int64) error {
	if fromAccountID == toAccountID {
		return ErrSameAccount
	}

	fromAccount, err := s.accountRepo.FindByID(ctx, fromAccountID)
	if err != nil {
		return fmt.Errorf("transfer_service.MakeTransfer find fromAccount: %w", err)
	}

	if fromAccount == nil {
		return ErrAccountNotFound
	}

	if fromAccount.UserID != userId {
		return ErrForbidden
	}

	toAccount, err := s.accountRepo.FindByID(ctx, toAccountID)
	if err != nil {
		return fmt.Errorf("transfer_service.MakeTransfer find toAccount: %w", err)
	}

	if toAccount == nil {
		return ErrAccountNotFound
	}

	err = fromAccount.Withdraw(amount)
	if err != nil {
		return ErrInsufficientFunds
	}

	err = toAccount.Deposit(amount)
	if err != nil {
		return err
	}

	err = postgres.WithTransaction(ctx, s.pool, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, debitAccountQuery, fromAccount.Balance, fromAccountID)
		if err != nil {
			return err
		}

		_, err = tx.Exec(ctx, creditAccountQuery, toAccount.Balance, toAccountID)
		if err != nil {
			return err
		}

		transferTx, err := domain.NewTransaction(fromAccountID, toAccountID, amount, domain.TransactionTypeTransfer, fmt.Sprintf("transfer from %d to %d", fromAccountID, toAccountID))
		if err != nil {
			return err
		}

		return s.transferRepo.Create(ctx, transferTx)
	})
	if err != nil {
		return fmt.Errorf("transfer_service.MakeTransfer: %w", err)
	}

	return nil
}

func (s *Service) GetHistory(ctx context.Context, userID int64, limit, offset int) ([]*domain.Transaction, error) {
	transactions, err := s.transferRepo.GetByUserID(ctx, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("transfer_service.GetHistory: %w", err)
	}

	return transactions, nil
}
