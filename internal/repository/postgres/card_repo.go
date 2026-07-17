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

type cardRepo struct {
	pool *pgxpool.Pool
}

func NewCardRepository(pool *pgxpool.Pool) repository.CardRepository {
	return &cardRepo{
		pool: pool,
	}
}

const createCardQuery = `
INSERT INTO cards (account_id, number_hash, number_enc, cvc_hash, expiry_enc, status, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING id, created_at, updated_at
`

func (r *cardRepo) Create(ctx context.Context, card *domain.Card) error {
	err := r.pool.QueryRow(
		ctx, createCardQuery,
		card.AccountID,
		card.Number,
		[]byte(card.Number),
		card.CVV,
		[]byte(card.ExpiresAt.Format("2006-01-02")),
		string(card.Status),
		card.CreatedAt,
		card.UpdatedAt,
	).Scan(&card.ID, &card.CreatedAt, &card.UpdatedAt)
	if err != nil {
		return fmt.Errorf("card_repo.Create: %w", err)
	}

	return nil
}

const getCardByIDQuery = `
SELECT id, account_id, number_hash, number_enc, cvc_hash, expiry_enc, status, created_at, updated_at
FROM cards
WHERE id = $1
`

func (r *cardRepo) FindByID(ctx context.Context, id int64) (*domain.Card, error) {
	var card domain.Card

	err := r.pool.QueryRow(ctx, getCardByIDQuery, id).Scan(
		&card.ID,
		&card.AccountID,
		&card.Number,
		&card.Number,
		&card.CVV,
		&card.ExpiresAt,
		&card.Status,
		&card.CreatedAt,
		&card.UpdatedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}

	if err != nil {
		return nil, fmt.Errorf("card_repo.FindByID: %w", err)
	}

	return &card, nil
}

const getCardsByAccountIDQuery = `
SELECT id, account_id, number_hash,	number_enc, cvc_hash, expiry_enc, status, created_at, updated_at
FROM cards
WHERE account_id = $1
ORDER BY created_at DESC
`

func (r *cardRepo) FindByAccountID(ctx context.Context, accountID int64) ([]*domain.Card, error) {
	rows, err := r.pool.Query(ctx, getCardsByAccountIDQuery, accountID)
	if err != nil {
		return nil, fmt.Errorf("card_repo.FindByAccountID: %w", err)
	}
	defer rows.Close()

	var cards []*domain.Card

	for rows.Next() {
		var card domain.Card

		err := rows.Scan(
			&card.ID,
			&card.AccountID,
			&card.Number,
			&card.Number,
			&card.CVV,
			&card.ExpiresAt,
			&card.Status,
			&card.CreatedAt,
			&card.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("card_repo.FindByAccountID: %w", err)
		}

		cards = append(cards, &card)
	}

	err = rows.Err()
	if err != nil {
		return nil, fmt.Errorf("card_repo.FindByAccountID: %w", err)
	}

	if cards == nil {
		return []*domain.Card{}, nil
	}

	return cards, nil
}

const updateCardStatusQuery = `
UPDATE cards
SET status = $1, updated_at = NOW()
WHERE id = $2
`

func (r *cardRepo) UpdateStatus(ctx context.Context, id int64, status domain.CardStatus) error {
	tag, err := r.pool.Exec(ctx, updateCardStatusQuery, string(status), id)
	if err != nil {
		return fmt.Errorf("card_repo.UpdateStatus: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return fmt.Errorf("card_repo.UpdateStatus: %w", repository.ErrNotFound)
	}

	return nil
}
