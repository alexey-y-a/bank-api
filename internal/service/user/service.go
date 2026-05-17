package user

import (
	"context"
	"fmt"
	"time"

	"github.com/alexey-y-a/bank-api/internal/domain"
	"github.com/alexey-y-a/bank-api/internal/repository"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type Service struct {
	repo      repository.UserRepository
	jwtSecret []byte
	jwtTTL    time.Duration
}

func NewService(repo repository.UserRepository, jwtSecret string, jwtTTLHours int) *Service {
	return &Service{
		repo:      repo,
		jwtSecret: []byte(jwtSecret),
		jwtTTL:    time.Duration(jwtTTLHours) * time.Hour,
	}
}

func (s *Service) Register(ctx context.Context, email, username, password string) (*domain.User, error) {
	existing, err := s.repo.FindByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("find by email: %w", err)
	}
	if existing != nil {
		return nil, ErrUserAlreadyExists
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	user, err := domain.NewUser(email, username, string(hashed))
	if err != nil {
		return nil, fmt.Errorf("create newUser in domain: %w", err)
	}

	err = s.repo.Create(ctx, user)
	if err != nil {
		return nil, fmt.Errorf("create user in db: %w", err)
	}

	user.Password = ""

	return user, nil
}

func (s *Service) Login(ctx context.Context, email, password string) (string, *domain.User, error) {
	user, err := s.repo.FindByEmail(ctx, email)
	if err != nil {
		return "", nil, fmt.Errorf("find by email: %w", err)
	}

	if user == nil {
		return "", nil, ErrInvalidCredentials
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err != nil {
		return "", nil, ErrInvalidCredentials
	}

	token := jwt.NewWithClaims(jwt.SigningMethodES256, jwt.RegisteredClaims{
		Subject:   fmt.Sprintf("%d", user.ID),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.jwtTTL)),
	})

	tokenString, err := token.SignedString(s.jwtSecret)
	if err != nil {
		return "", nil, fmt.Errorf("sign jwt token: %w", err)
	}

	user.Password = ""
	return tokenString, user, nil
}
