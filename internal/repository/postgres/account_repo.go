package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/alexey-y-a/bank-api/internal/domain"
	"github.com/alexey-y-a/bank-api/internal/repository"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type accountRepo struct {
	pool *pgxpool.Pool
}

func NewAccountRepository(pool *pgxpool.Pool) repository.AccountRepository {
	return &accountRepo{
		pool: pool,
	}
}

const createAccountQuery = `
INSERT INTO accounts (user_id, balance, currency, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, created_at, updated_at
`

func (r *accountRepo) Create(ctx context.Context, account *domain.Account) error {
	err := r.pool.QueryRow(ctx, createAccountQuery,
		account.UserID,
		account.Balance,
		account.Currency,
		account.CreatedAt,
		account.UpdatedAt).Scan(&account.ID, &account.CreatedAt, &account.UpdatedAt)
	if err != nil {
		return fmt.Errorf("account_repo.Create: %w", err)
	}

	return nil
}

const getAccountByIDQuery = `
SELECT id, user_id, balance, currency, created_at, updated_at
FROM accounts
WHERE id = $1
`

func (r *accountRepo) FindByID(ctx context.Context, id int64) (*domain.Account, error) {
	var account domain.Account
	err := r.pool.QueryRow(ctx, getAccountByIDQuery, id).Scan(
		&account.ID,
		&account.UserID,
		&account.Balance,
		&account.Currency,
		&account.CreatedAt,
		&account.UpdatedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}

	if err != nil {
		return nil, fmt.Errorf("account_repo.FindByID: %w", err)
	}

	return &account, nil
}

const getAccountsByUserIDQuery = `
SELECT id, user_id, balance, currency, created_at, updated_at
FROM accounts
WHERE user_id = $1
ORDER BY created_at DESC
`

func (r *accountRepo) FindByUserID(ctx context.Context, userID int64) ([]*domain.Account, error) {
	rows, err := r.pool.Query(ctx, getAccountsByUserIDQuery, userID)
	if err != nil {
		return nil, fmt.Errorf("account_repo.FindByUserID: %w", err)
	}

	defer rows.Close()

	var accounts []*domain.Account

	for rows.Next() {
		var account domain.Account
		err := rows.Scan(
			&account.ID,
			&account.UserID,
			&account.Balance,
			&account.Currency,
			&account.CreatedAt,
			&account.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("account_repo.FindByUserID scan: %w", err)
		}

		accounts = append(accounts, &account)
	}

	err = rows.Err()
	if err != nil {
		return nil, fmt.Errorf("account_repo.FindByUserID rows.Err: %w", err)
	}

	if accounts == nil {
		return []*domain.Account{}, nil
	}

	return accounts, nil
}

const updateBalanceQuery = `
UPDATE accounts
SET balance = $1, updated_at = NOW()
WHERE id = $2
`

func (r *accountRepo) UpdateBalance(ctx context.Context, id int64, balance int64) error {
	tag, err := r.pool.Exec(ctx, updateBalanceQuery, balance, id)
	if err != nil {
		return fmt.Errorf("account_repo.UpdateBalance: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return fmt.Errorf("account_repo.UpdateBalance: %w", repository.ErrNotFound)
	}
	return nil
}
