package transfer

import (
	"context"
	"errors"
	"testing"

	"github.com/alexey-y-a/bank-api/internal/domain"
	"github.com/stretchr/testify/require"
)

type mockAccountRepo struct {
	findByIDFn      func(ctx context.Context, id int64) (*domain.Account, error)
	updateBalanceFn func(ctx context.Context, id int64, balance int64) error
}

func (m *mockAccountRepo) Create(ctx context.Context, account *domain.Account) error {
	return nil
}

func (m *mockAccountRepo) FindByID(ctx context.Context, id int64) (*domain.Account, error) {
	return m.findByIDFn(ctx, id)
}

func (m *mockAccountRepo) FindByUserID(ctx context.Context, userID int64) ([]*domain.Account, error) {
	return nil, nil
}

func (m *mockAccountRepo) UpdateBalance(ctx context.Context, id int64, balance int64) error {
	if m.updateBalanceFn != nil {
		return m.updateBalanceFn(ctx, id, balance)
	}

	return nil
}

type mockTransferRepo struct {
	createFn      func(ctx context.Context, tx *domain.Transaction) error
	getByUserIDFn func(ctx context.Context, userID int64, limit, offset int) ([]*domain.Transaction, error)
}

func (m *mockTransferRepo) Create(ctx context.Context, tx *domain.Transaction) error {
	return m.createFn(ctx, tx)
}

func (m *mockTransferRepo) GetByAccountID(ctx context.Context, accountID int64, limit, offset int) ([]*domain.Transaction, error) {
	return nil, nil
}

func (m *mockTransferRepo) GetByUserID(ctx context.Context, userID int64, limit, offset int) ([]*domain.Transaction, error) {
	return m.getByUserIDFn(ctx, userID, limit, offset)
}

func TestDeposit(t *testing.T) {
	tests := []struct {
		name        string
		accountID   int64
		userID      int64
		amount      int64
		setupMocks  func(accountRepo *mockAccountRepo, transferRepo *mockTransferRepo)
		expectedErr error
	}{
		{
			name:      "успешное пополнение",
			accountID: 1,
			userID:    5,
			amount:    5000,
			setupMocks: func(accountRepo *mockAccountRepo, transferRepo *mockTransferRepo) {
				accountRepo.findByIDFn = func(ctx context.Context, id int64) (*domain.Account, error) {
					return &domain.Account{ID: 1, UserID: 5, Balance: 10000}, nil
				}
				accountRepo.updateBalanceFn = func(ctx context.Context, id int64, balance int64) error {
					return nil
				}
				transferRepo.createFn = func(ctx context.Context, tx *domain.Transaction) error {
					return nil
				}
			},
			expectedErr: nil,
		},
		{
			name:      "ошибка: счет не найден",
			accountID: 999,
			userID:    5,
			amount:    5000,
			setupMocks: func(accountRepo *mockAccountRepo, transferRepo *mockTransferRepo) {
				accountRepo.findByIDFn = func(ctx context.Context, id int64) (*domain.Account, error) {
					return nil, nil
				}
			},
			expectedErr: ErrAccountNotFound,
		},
		{
			name:      "ошибка: счет не принадлежит пользователю",
			accountID: 1,
			userID:    5,
			amount:    5000,
			setupMocks: func(accountRepo *mockAccountRepo, transferRepo *mockTransferRepo) {
				accountRepo.findByIDFn = func(ctx context.Context, id int64) (*domain.Account, error) {
					return &domain.Account{ID: 1, UserID: 99, Balance: 10000}, nil
				}
			},
			expectedErr: ErrForbidden,
		},
		{
			name:      "ошибка: сбой БД при поиске счета",
			accountID: 1,
			userID:    5,
			amount:    5000,
			setupMocks: func(accountRepo *mockAccountRepo, transferRepo *mockTransferRepo) {
				accountRepo.findByIDFn = func(ctx context.Context, id int64) (*domain.Account, error) {
					return nil, errors.New("db connection failed")
				}
			},
			expectedErr: errors.New("db connection failed"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			accountRepo := &mockAccountRepo{}
			transferRepo := &mockTransferRepo{}

			tt.setupMocks(accountRepo, transferRepo)

			svc := NewService(accountRepo, transferRepo, nil)

			err := svc.Deposit(context.Background(), tt.accountID, tt.userID, tt.amount)
			if tt.expectedErr != nil {
				require.Error(t, tt.expectedErr, err, "должна быть ошибка")
				return
			}

			require.NoError(t, err, "не должно быть ошибки")
		})
	}
}

func TestGetHistory(t *testing.T) {
	tests := []struct {
		name        string
		userID      int64
		setupMocks  func(transferRepo *mockTransferRepo)
		expectedLen int
		expectedErr error
	}{
		{
			name:   "успешное получение истории с транзакциями",
			userID: 5,
			setupMocks: func(transferRepo *mockTransferRepo) {
				transferRepo.getByUserIDFn = func(ctx context.Context, userID int64, limit, offset int) ([]*domain.Transaction, error) {
					return []*domain.Transaction{
						{ID: 1, Amount: 5000},
						{ID: 2, Amount: 3000},
					}, nil
				}
			},
			expectedLen: 2,
			expectedErr: nil,
		},
		{
			name:   "пустая история",
			userID: 5,
			setupMocks: func(transferRepo *mockTransferRepo) {
				transferRepo.getByUserIDFn = func(ctx context.Context, userID int64, limit, offset int) ([]*domain.Transaction, error) {
					return []*domain.Transaction{}, nil
				}
			},
			expectedLen: 0,
			expectedErr: nil,
		},
		{
			name:   "сбой БД при запросе истории",
			userID: 5,
			setupMocks: func(transferRepo *mockTransferRepo) {
				transferRepo.getByUserIDFn = func(ctx context.Context, userID int64, limit, offset int) ([]*domain.Transaction, error) {
					return nil, errors.New("db timeout")
				}
			},
			expectedLen: 0,
			expectedErr: errors.New("db timeout"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transferRepo := &mockTransferRepo{}
			tt.setupMocks(transferRepo)

			svc := NewService(&mockAccountRepo{}, transferRepo, nil)

			transactions, err := svc.GetHistory(context.Background(), tt.userID, 10, 0)

			if tt.expectedErr != nil {
				require.Error(t, err, "должна быть ошибка")
				return
			}

			require.NoError(t, err, "не должно быть ошибки")
			require.Len(t, transactions, tt.expectedLen, "количество транзакций должно совпадать")
		})
	}
}
