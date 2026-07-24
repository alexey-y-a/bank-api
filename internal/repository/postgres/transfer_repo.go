package postgres

import (
	"context"
	"fmt"

	"github.com/alexey-y-a/bank-api/internal/domain"
	"github.com/alexey-y-a/bank-api/internal/repository"
	"github.com/jackc/pgx/v5/pgxpool"
)

type transferRepo struct {
	pool *pgxpool.Pool
}

func NewTransferRepo(pool *pgxpool.Pool) repository.TransferRepository {
	return &transferRepo{
		pool: pool,
	}
}

const createTransactionQuery = `
INSERT INTO transactions (from_account_id, to_account_id, amount, type, status, description, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id, created_at
`

func (r *transferRepo) Create(ctx context.Context, tx *domain.Transaction) error {
	err := r.pool.QueryRow(
		ctx, createTransactionQuery,
		tx.FromAccountID,
		tx.ToAccountID,
		tx.Amount,
		string(tx.Type),
		string(tx.Status),
		tx.Description,
		tx.CreatedAt,
	).Scan(&tx.ID, &tx.CreatedAt)
	if err != nil {
		return fmt.Errorf("transfer_repo.Create: %w", err)
	}

	return nil
}

const getTransactionsByAccountIDQuery = `
SELECT id, from_account_id, to_account_id, amount, type, status, description, created_at
FROM transactions
WHERE from_account_id = $1 OR to_account_id = $1
ORDER BY created_at DESC 
LIMIT $2 OFFSET $3
`

func (r *transferRepo) GetByAccountID(ctx context.Context, accountID int64, limit, offset int) ([]*domain.Transaction, error) {
	rows, err := r.pool.Query(ctx, getTransactionsByAccountIDQuery, accountID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("transfer_repo.GetByAccountID: %w", err)
	}

	defer rows.Close()

	var transactions []*domain.Transaction
	for rows.Next() {
		var tx domain.Transaction
		err := rows.Scan(
			&tx.ID,
			&tx.FromAccountID,
			&tx.ToAccountID,
			&tx.Amount,
			&tx.Type,
			&tx.Status,
			&tx.Description,
			&tx.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("transfer_repo.GetByAccountID scan: %w", err)
		}

		transactions = append(transactions, &tx)
	}

	err = rows.Err()
	if err != nil {
		return nil, fmt.Errorf("transfer_repo.GetByAccountID rows: %w", err)
	}

	if transactions == nil {
		return []*domain.Transaction{}, nil
	}

	return transactions, nil
}

const getTransactionsByUserIDQuery = `
SELECT t.id, t.from_account_id, t.to_account_id, t.amount, t.type, t.status, t.description, t.created_at
FROM transactions t
JOIN accounts as a ON (t.from_account_id = a.id OR t.to_account_id = a.id)
WHERE a.user_id = $1
ORDER BY t.created_at DESC
LIMIT $2 OFFSET $3
`

func (r *transferRepo) GetByUserID(ctx context.Context, userID int64, limit, offset int) ([]*domain.Transaction, error) {
	rows, err := r.pool.Query(ctx, getTransactionsByUserIDQuery, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("transfer_repo.GetByUserID: %w", err)
	}

	defer rows.Close()

	var transactions []*domain.Transaction
	for rows.Next() {
		var tx domain.Transaction
		err := rows.Scan(
			&tx.ID,
			&tx.FromAccountID,
			&tx.ToAccountID,
			&tx.Amount,
			&tx.Type,
			&tx.Status,
			&tx.Description,
			&tx.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("transfer_repo.GetByUserID scan: %w", err)
		}

		transactions = append(transactions, &tx)
	}

	err = rows.Err()
	if err != nil {
		return nil, fmt.Errorf("transfer_repo.GetByUserID rows: %w", err)
	}

	if transactions == nil {
		return []*domain.Transaction{}, nil
	}

	return transactions, nil
}
