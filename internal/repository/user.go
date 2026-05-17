package repository

import (
	"context"

	"github.com/alexey-y-a/bank-api/internal/domain"
)

//go:generate mockgen -source=$GOFILE -destination=mocks/user_repo_mock.gen.go -package=mocks

type UserRepository interface {
	Create(ctx context.Context, user *domain.User) error
	FindByEmail(ctx context.Context, email string) (*domain.User, error)
	FindByID(ctx context.Context, id int64) (*domain.User, error)
}
