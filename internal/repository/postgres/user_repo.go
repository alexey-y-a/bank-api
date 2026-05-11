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

type userRepo struct {
	pool *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) repository.UserRepository {
	return &userRepo{
		pool: pool,
	}
}

const createUserQuery = `
INSERT INTO users (email, username, password_hash, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, created_at, updated_at
`

func (r *userRepo) Create(ctx context.Context, user *domain.User) error {
	err := r.pool.QueryRow(ctx, createUserQuery, user.Email, user.Username, user.Password, user.CreatedAt, user.UpdatedAt).
		Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return fmt.Errorf("user create: %w", err)
	}

	return nil
}

const getUserByEmailQuery = `
SELECT id, email, username, password_hash, created_at, updated_at
FROM users
WHERE email = $1
`

func (r *userRepo) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	var u domain.User

	err := r.pool.QueryRow(ctx, getUserByEmailQuery, email).
		Scan(&u.ID, &u.Email, &u.Username, &u.Password, &u.CreatedAt, &u.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}

	if err != nil {
		return nil, fmt.Errorf("user find by email: %w", err)
	}

	return &u, nil
}

const getUserByIDQuery = `
SELECT id, email, username, password_hash, created_at, updated_at
FROM users
WHERE id = $1
`

func (r *userRepo) FindByID(ctx context.Context, id int64) (*domain.User, error) {
	var u domain.User

	err := r.pool.QueryRow(ctx, getUserByIDQuery, id).
		Scan(&u.ID, &u.Email, &u.Username, &u.Password, &u.CreatedAt, &u.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}

	if err != nil {
		return nil, fmt.Errorf("user find by id: %w", err)
	}

	return &u, nil
}
