package account

import (
	"context"
	"errors"
	"testing"

	"github.com/alexey-y-a/bank-api/internal/domain"
	"github.com/alexey-y-a/bank-api/internal/repository"
	"github.com/stretchr/testify/require"
)

type mockRepo struct {
	createFn        func(ctx context.Context, account *domain.Account) error
	findByIDFn      func(ctx context.Context, id int64) (*domain.Account, error)
	findByUserIDFn  func(ctx context.Context, userID int64) ([]*domain.Account, error)
	updateBalanceFn func(ctx context.Context, id int64, balance int64) error
}

func (m *mockRepo) Create(ctx context.Context, account *domain.Account) error {
	return m.createFn(ctx, account)
}

func (m *mockRepo) FindByID(ctx context.Context, id int64) (*domain.Account, error) {
	return m.findByIDFn(ctx, id)
}

func (m *mockRepo) FindByUserID(ctx context.Context, userID int64) ([]*domain.Account, error) {
	return m.findByUserIDFn(ctx, userID)
}

func (m *mockRepo) UpdateBalance(ctx context.Context, id int64, balance int64) error {
	return m.updateBalanceFn(ctx, id, balance)
}

var _ repository.AccountRepository = (*mockRepo)(nil)

func TestAccountService_Create(t *testing.T) {
	repo := &mockRepo{
		createFn: func(ctx context.Context, account *domain.Account) error {
			account.ID = 1
			return nil
		},
	}

	svc := NewService(repo)

	t.Run("успешное создание счета", func(t *testing.T) {
		account, err := svc.Create(context.Background(), 1, "RUB")

		require.NoError(t, err, "создание счета не должно возвращать ошибку")
		require.Equal(t, int64(1), account.ID, "ID должен быть заполнен")
		require.Equal(t, int64(1), account.UserID, "владелец счета должен быть 1")
		require.Equal(t, int64(0), account.Balance, "у нового счета баланс 0")
		require.Equal(t, "RUB", account.Currency, "валюта должна быть RUB")
	})

	t.Run("ошибка репозитория при создании", func(t *testing.T) {
		repo.createFn = func(ctx context.Context, account *domain.Account) error {
			return errors.New("db connection failed")
		}

		_, err := svc.Create(context.Background(), 1, "RUB")
		require.Error(t, err, "должно вернуть ошибку")
		require.Contains(t, err.Error(), "db connection failed")
	})
}

func TestAccountService_GetByID(t *testing.T) {
	t.Run("счет найден и принадлежит пользователю", func(t *testing.T) {
		repo := &mockRepo{
			findByIDFn: func(ctx context.Context, id int64) (*domain.Account, error) {
				return &domain.Account{ID: 1, UserID: 5, Balance: 1000}, nil
			},
		}

		svc := NewService(repo)

		account, err := svc.GetByID(context.Background(), 1, 5)

		require.NoError(t, err, "должны получить счет")
		require.Equal(t, int64(1), account.ID)
		require.Equal(t, int64(5), account.UserID)
	})

	t.Run("счет найден", func(t *testing.T) {
		repo := &mockRepo{
			findByIDFn: func(ctx context.Context, id int64) (*domain.Account, error) {
				return nil, nil
			},
		}
		svc := NewService(repo)

		_, err := svc.GetByID(context.Background(), 999, 5)

		require.ErrorIs(t, err, ErrAccountNotFound, "должна быть ошибка account not found")
	})

	t.Run("доступ к чужому счету запрещен", func(t *testing.T) {
		repo := &mockRepo{
			findByIDFn: func(ctx context.Context, id int64) (*domain.Account, error) {
				return &domain.Account{ID: 1, UserID: 5}, nil
			},
		}

		svc := NewService(repo)

		_, err := svc.GetByID(context.Background(), 1, 2)

		require.ErrorIs(t, err, ErrForbidden, "доступ к чужому счету должен быть запрещен")
	})

	t.Run("ошибка репозитория", func(t *testing.T) {
		repo := &mockRepo{
			findByIDFn: func(ctx context.Context, id int64) (*domain.Account, error) {
				return nil, errors.New("db timeout")
			},
		}

		svc := NewService(repo)

		_, err := svc.GetByID(context.Background(), 1, 5)

		require.Error(t, err, "должна быть ошибка")
		require.Contains(t, err.Error(), "db timeout")
	})
}

func TestAccountService_Deposit(t *testing.T) {
	t.Run("успешное пополнение", func(t *testing.T) {
		repo := &mockRepo{
			findByIDFn: func(ctx context.Context, id int64) (*domain.Account, error) {
				return &domain.Account{ID: 1, UserID: 5, Balance: 0}, nil
			},
			updateBalanceFn: func(ctx context.Context, id int64, balance int64) error {
				require.Equal(t, int64(5000), balance, "баланс должен быть 5000 после пополнения")
				return nil
			},
		}

		svc := NewService(repo)

		account, err := svc.Deposit(context.Background(), 1, 5, 5000)

		require.NoError(t, err)
		require.Equal(t, int64(5000), account.Balance, "баланс должен увеличиться на 5000")
	})

	t.Run("пополнение на 0 - ошибка", func(t *testing.T) {
		repo := &mockRepo{
			findByIDFn: func(ctx context.Context, id int64) (*domain.Account, error) {
				return &domain.Account{ID: 1, UserID: 5, Balance: 1000}, nil
			},
			updateBalanceFn: func(ctx context.Context, id int64, balance int64) error {
				t.Error("updateBalance не должен вызываться при ошибке")
				return nil
			},
		}

		svc := NewService(repo)

		_, err := svc.Deposit(context.Background(), 1, 5, 0)

		require.ErrorIs(t, err, domain.ErrInvalidAmount, "пополнение на 0 должно вернуть ошибку")
	})
}

func TestAccountService_Withdraw(t *testing.T) {
	t.Run("успешное списание", func(t *testing.T) {
		repo := &mockRepo{
			findByIDFn: func(ctx context.Context, id int64) (*domain.Account, error) {
				return &domain.Account{ID: 1, UserID: 5, Balance: 10000}, nil
			},
			updateBalanceFn: func(ctx context.Context, id int64, balance int64) error {
				require.Equal(t, int64(3000), balance, "баланс должен быть 3000 после списания 7000")
				return nil
			},
		}

		svc := NewService(repo)

		account, err := svc.Withdraw(context.Background(), 1, 5, 7000)

		require.NoError(t, err)
		require.Equal(t, int64(3000), account.Balance, "баланс должен уменьшиться на 7000")
	})

	t.Run("недостаточно средств", func(t *testing.T) {
		repo := &mockRepo{
			findByIDFn: func(ctx context.Context, id int64) (*domain.Account, error) {
				return &domain.Account{ID: 1, UserID: 5, Balance: 1000}, nil
			},
			updateBalanceFn: func(ctx context.Context, id int64, balance int64) error {
				t.Error("updateBalance не должен вызываться при ошибке")
				return nil
			},
		}

		svc := NewService(repo)

		_, err := svc.Withdraw(context.Background(), 1, 5, 5000)

		require.ErrorIs(t, err, domain.ErrInsufficientFunds, "должна быть ошибка insufficient funds")
	})
}
