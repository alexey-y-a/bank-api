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

type creditRepo struct {
	pool *pgxpool.Pool
}

func NewCreditRepo(pool *pgxpool.Pool) repository.CreditRepository {
	return &creditRepo{
		pool: pool,
	}
}

const createCreditQuery = `
INSERT INTO credits (account_id, amount, rate, term_months, status, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id, created_at, updated_at
`

func (r *creditRepo) CreateCredit(ctx context.Context, credit *domain.Credit) error {
	err := r.pool.QueryRow(ctx, createCreditQuery,
		credit.AccountID,
		credit.Amount,
		credit.Rate,
		credit.TermMonths,
		string(credit.Status),
		credit.CreatedAt,
		credit.UpdatedAt).Scan(&credit.ID, &credit.CreatedAt, &credit.UpdatedAt)
	if err != nil {
		return fmt.Errorf("credit_repo.CreateCredit: %w", err)
	}

	return nil
}

const getCreditByIDQuery = `
SELECT id, account_id, amount, rate, term_months, status, created_at, updated_at
FROM credits
WHERE id = $1
`

func (r *creditRepo) FindByID(ctx context.Context, id int64) (*domain.Credit, error) {
	var credit domain.Credit

	err := r.pool.QueryRow(ctx, getCreditByIDQuery, id).Scan(
		&credit.ID,
		&credit.AccountID,
		&credit.Amount,
		&credit.Rate,
		&credit.TermMonths,
		&credit.Status,
		&credit.CreatedAt,
		&credit.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}

	if err != nil {
		return nil, fmt.Errorf("credit_repo.FindByID: %w", err)
	}

	return &credit, nil
}

const getCreditsByAccountIDQuery = `
SELECT id, account_id, amount, rate, term_months, status, created_at, updated_at
FROM credits
WHERE account_id = $1
ORDER BY created_at DESC 
`

func (r *creditRepo) FindByAccountID(ctx context.Context, accountID int64) ([]*domain.Credit, error) {
	rows, err := r.pool.Query(ctx, getCreditsByAccountIDQuery, accountID)
	if err != nil {
		return nil, fmt.Errorf("credit_repo.FindByAccountID: %w", err)
	}

	defer rows.Close()

	var credits []*domain.Credit

	for rows.Next() {
		var credit domain.Credit
		err := rows.Scan(
			&credit.ID,
			&credit.AccountID,
			&credit.Amount,
			&credit.Rate,
			&credit.TermMonths,
			&credit.Status,
			&credit.CreatedAt,
			&credit.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("credit_repo.FindByAccountID scan: %w", err)
		}

		credits = append(credits, &credit)
	}

	err = rows.Err()
	if err != nil {
		return nil, fmt.Errorf("credit_repo.FindByAccountID rows: %w", err)
	}

	if credits == nil {
		return []*domain.Credit{}, nil
	}

	return credits, nil
}

const updateCreditStatusQuery = `
UPDATE credits
SET status = $1, updated_at = NOW()
WHERE id = $2
`

func (r *creditRepo) UpdateStatus(ctx context.Context, id int64, status domain.CreditStatus) error {
	tag, err := r.pool.Exec(ctx, updateCreditStatusQuery, string(status), id)
	if err != nil {
		return fmt.Errorf("credit_repo.UpdateStatus: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return fmt.Errorf("credit_repo.UpdateStatus: %w", repository.ErrNotFound)
	}

	return nil
}

const createScheduleItemQuery = `
INSERT INTO payment_schedules (credit_id, payment_date, principal, interest, total, remaining_balance, status)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id
`

func (r *creditRepo) CreateScheduleItem(ctx context.Context, item *domain.CreditScheduleItem) error {
	err := r.pool.QueryRow(ctx, createScheduleItemQuery,
		item.CreditID,
		item.PaymentDate,
		item.Principal,
		item.Interest,
		item.Total,
		item.RemainingBalance,
		string(item.Status)).Scan(&item.ID)
	if err != nil {
		return fmt.Errorf("credit_repo.CreateScheduleItem: %w", err)
	}

	return nil
}

const getScheduleByCreditIDQuery = `
SELECT id, credit_id, payment_date, principal, interest, total, remaining_balance, status
FROM payment_schedules
WHERE credit_id = $1
ORDER BY payment_date ASC
`

func (r *creditRepo) FindScheduleByCreditID(ctx context.Context, creditID int64) ([]*domain.CreditScheduleItem, error) {
	rows, err := r.pool.Query(ctx, getScheduleByCreditIDQuery, creditID)
	if err != nil {
		return nil, fmt.Errorf("credit_repo.FindScheduleByCreditID: %w", err)
	}

	defer rows.Close()

	var items []*domain.CreditScheduleItem

	for rows.Next() {
		var item domain.CreditScheduleItem
		err := rows.Scan(
			&item.ID,
			&item.CreditID,
			&item.PaymentDate,
			&item.Principal,
			&item.Interest,
			&item.Total,
			&item.RemainingBalance,
			&item.Status,
		)
		if err != nil {
			return nil, fmt.Errorf("credit_repo.FindScheduleByCreditID scan: %w", err)
		}

		items = append(items, &item)
	}

	err = rows.Err()
	if err != nil {
		return nil, fmt.Errorf("credit_repo.FindScheduleByCreditID rows: %w", err)
	}

	if items == nil {
		return []*domain.CreditScheduleItem{}, nil
	}

	return items, nil
}
