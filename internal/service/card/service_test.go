package card

import (
	"context"
	"testing"
	"time"

	"github.com/alexey-y-a/bank-api/internal/crypto"
	"github.com/alexey-y-a/bank-api/internal/domain"
	"github.com/alexey-y-a/bank-api/internal/repository"
	"github.com/stretchr/testify/require"
)

type mockCardRepo struct {
	createFn          func(ctx context.Context, card *domain.Card) error
	findByIDFn        func(ctx context.Context, id int64) (*domain.Card, error)
	findByAccountIDFn func(ctx context.Context, accountID int64) ([]*domain.Card, error)
	updateStatusFn    func(ctx context.Context, id int64, status domain.CardStatus) error
}

func (m *mockCardRepo) Create(ctx context.Context, card *domain.Card) error {
	return m.createFn(ctx, card)
}

func (m *mockCardRepo) FindByID(ctx context.Context, id int64) (*domain.Card, error) {
	return m.findByIDFn(ctx, id)
}

func (m *mockCardRepo) FindByAccountID(ctx context.Context, accountID int64) ([]*domain.Card, error) {
	return m.findByAccountIDFn(ctx, accountID)
}

func (m *mockCardRepo) UpdateStatus(ctx context.Context, id int64, status domain.CardStatus) error {
	return m.updateStatusFn(ctx, id, status)
}

var _ repository.CardRepository = (*mockCardRepo)(nil)

func TestService_IssueCard(t *testing.T) {
	t.Run("успешный выпуск карты", func(t *testing.T) {
		enc := crypto.NewCardEncryptor("test-aes-secret-32byte!!", "test-hmac-secret")

		cardRepo := &mockCardRepo{
			createFn: func(ctx context.Context, card *domain.Card) error {
				card.ID = 1
				return nil
			},
		}

		accountRepo := &mockAccountRepo{
			findByIDFn: func(ctx context.Context, id int64) (*domain.Account, error) {
				return &domain.Account{ID: id, UserID: 5}, nil
			},
		}

		svc := NewService(enc, cardRepo, accountRepo)

		card, err := svc.IssueCard(context.Background(), 1, 5)

		require.NoError(t, err, "не должно быть ошибки")
		require.NotNil(t, card, "карта должна быть создана")
		require.Equal(t, int64(1), card.ID, "ID карты должен быть заполнен")
	})

	t.Run("ошибка: счет не принадлежит пользователю", func(t *testing.T) {
		enc := crypto.NewCardEncryptor("test-aes-secret-32byte!!", "test-hmac-secret")
		cardRepo := &mockCardRepo{}
		accountRepo := &mockAccountRepo{
			findByIDFn: func(ctx context.Context, id int64) (*domain.Account, error) {
				return &domain.Account{ID: id, UserID: 99}, nil
			},
		}

		svc := NewService(enc, cardRepo, accountRepo)

		_, err := svc.IssueCard(context.Background(), 1, 5)

		require.ErrorIs(t, err, ErrCardNotFound, "должна быть ошибка not found")
	})
}

type mockAccountRepo struct {
	findByIDFn      func(ctx context.Context, id int64) (*domain.Account, error)
	findByUserIDFn  func(ctx context.Context, userID int64) ([]*domain.Account, error)
	updateBalanceFn func(ctx context.Context, id int64, balance int64) error
}

func (m *mockAccountRepo) Create(ctx context.Context, account *domain.Account) error {
	return nil
}

func (m *mockAccountRepo) FindByID(ctx context.Context, id int64) (*domain.Account, error) {
	return m.findByIDFn(ctx, id)
}

func (m *mockAccountRepo) FindByUserID(ctx context.Context, userID int64) ([]*domain.Account, error) {
	if m.findByIDFn == nil {
		return m.findByUserIDFn(ctx, userID)
	}

	return []*domain.Account{{ID: 1, UserID: userID}}, nil
}

func (m *mockAccountRepo) UpdateBalance(ctx context.Context, id int64, balance int64) error {
	if m.updateBalanceFn == nil {
		return m.updateBalanceFn(ctx, id, balance)
	}

	return nil
}

func TestService_BlockCard(t *testing.T) {
	t.Run("успешная блокировка", func(t *testing.T) {
		enc := crypto.NewCardEncryptor("test-aes-secret-32byte!!", "test-hmac-secret")
		cardRepo := &mockCardRepo{
			findByIDFn: func(ctx context.Context, id int64) (*domain.Card, error) {
				return &domain.Card{ID: id, AccountID: 1}, nil
			},

			updateStatusFn: func(ctx context.Context, id int64, status domain.CardStatus) error {
				return nil
			},
		}

		accountRepo := &mockAccountRepo{
			findByIDFn: func(ctx context.Context, id int64) (*domain.Account, error) {
				return &domain.Account{ID: id, UserID: 5}, nil
			},
		}

		svc := NewService(enc, cardRepo, accountRepo)

		err := svc.BlockCard(context.Background(), 1, 5)

		require.NoError(t, err, "успешная блокировка без ошибок")
	})
}

func TestService_PayWithCard(t *testing.T) {
	t.Run("успешная оплата", func(t *testing.T) {
		enc := crypto.NewCardEncryptor("test-aes-secret-32byte!!", "test-hmac-secret")

		cvvHash, err := enc.HashCVV("123")
		require.NoError(t, err)

		cardRepo := &mockCardRepo{
			findByIDFn: func(ctx context.Context, id int64) (*domain.Card, error) {
				return &domain.Card{
					ID:        id,
					AccountID: 1,
					CVV:       cvvHash,
					Status:    domain.CardStatusActive,
					ExpiresAt: time.Now().AddDate(1, 0, 0),
				}, nil
			},
		}

		accountRepo := &mockAccountRepo{
			findByIDFn: func(ctx context.Context, id int64) (*domain.Account, error) {
				return &domain.Account{ID: id, UserID: 5, Balance: 10000}, nil
			},

			updateBalanceFn: func(ctx context.Context, id int64, balance int64) error {
				return nil
			},
		}

		svc := NewService(enc, cardRepo, accountRepo)
		err = svc.PayWithCard(context.Background(), 1, 5, "123", 5000)

		require.NoError(t, err, "оплата должна пройти успешно")
	})

	t.Run("ошибка: неверный CVV", func(t *testing.T) {
		enc := crypto.NewCardEncryptor("test-aes-secret-32byte!!", "test-hmac-secret")

		cvvHash, err := enc.HashCVV("123")
		require.NoError(t, err)

		cardRepo := &mockCardRepo{
			findByIDFn: func(ctx context.Context, id int64) (*domain.Card, error) {
				return &domain.Card{
					ID:     id,
					CVV:    cvvHash,
					Status: domain.CardStatusActive,
				}, nil
			},
		}

		accountRepo := &mockAccountRepo{
			findByIDFn: func(ctx context.Context, id int64) (*domain.Account, error) {
				return &domain.Account{ID: id, UserID: 5, Balance: 10000}, nil
			},
		}

		svc := NewService(enc, cardRepo, accountRepo)
		err = svc.PayWithCard(context.Background(), 1, 5, "999", 5000)

		require.ErrorIs(t, err, ErrInvalidCVV, "должна быть ошибка invalid cvv")
	})
}
