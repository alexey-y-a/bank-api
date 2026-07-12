package user

import (
	"context"
	"fmt"
	"time"

	"github.com/alexey-y-a/bank-api/internal/domain"
	"github.com/alexey-y-a/bank-api/internal/repository"
	goredis "github.com/alexey-y-a/bank-api/internal/repository/redis"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type Service struct {
	repo      repository.UserRepository
	cache     *goredis.UserCache
	jwtSecret []byte
	jwtTTL    time.Duration
}

func NewService(repo repository.UserRepository, cache *goredis.UserCache, jwtSecret string, jwtTTLHours int) *Service {
	return &Service{
		repo:      repo,
		cache:     cache,
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

	if s.cache != nil {
		err = s.cache.Set(ctx, user.ID, user)
		if err != nil {
			fmt.Printf("warn: failed to cache user %d: %v\n", user.ID, err)
		}
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

	if s.cache != nil {
		_ = s.cache.Set(ctx, user.ID, user)
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
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

func (s *Service) FindByID(ctx context.Context, id int64) (*domain.User, error) {
	if s.cache != nil {
		cached, err := s.cache.Get(ctx, id)
		if err == nil && cached != nil {
			return cached, nil
		}
		if err != nil {
			fmt.Printf("warn: cache get failed fo user %d: %v\n", id, err)
		}
	}

	user, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("find by id: %w", err)
	}

	if user == nil {
		return nil, ErrUserNotFound
	}

	if s.cache != nil {
		_ = s.cache.Set(ctx, user.ID, user)
	}

	return user, nil
}
