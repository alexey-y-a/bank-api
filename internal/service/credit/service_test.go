package credit

import (
	"context"
	"testing"

	"github.com/alexey-y-a/bank-api/internal/domain"
	"github.com/stretchr/testify/require"
)

type mockCreditRepo struct {
	createCreditFn             func(ctx context.Context, credit *domain.Credit) error
	findByIDFn                 func(ctx context.Context, id int64) (*domain.Credit, error)
	findScheduleByCreditIDFn   func(ctx context.Context, creditID int64) ([]*domain.CreditScheduleItem, error)
	createScheduleItemFn       func(ctx context.Context, item *domain.CreditScheduleItem) error
	updateScheduleItemStatusFn func(ctx context.Context, itemID int64, status domain.PaymentStatus) error
}

func (m *mockCreditRepo) CreateCredit(ctx context.Context, credit *domain.Credit) error {
	return m.createCreditFn(ctx, credit)
}

func (m *mockCreditRepo) FindByID(ctx context.Context, id int64) (*domain.Credit, error) {
	return m.findByIDFn(ctx, id)
}

func (m *mockCreditRepo) FindByAccountID(ctx context.Context, accountID int64) ([]*domain.Credit, error) {
	return nil, nil
}

func (m *mockCreditRepo) UpdateStatus(ctx context.Context, itemID int64, status domain.CreditStatus) error {
	return nil
}

func (m *mockCreditRepo) CreateScheduleItem(ctx context.Context, item *domain.CreditScheduleItem) error {
	return m.createScheduleItemFn(ctx, item)
}

func (m *mockCreditRepo) FindScheduleByCreditID(ctx context.Context, creditID int64) ([]*domain.CreditScheduleItem, error) {
	return m.findScheduleByCreditIDFn(ctx, creditID)
}

func (m *mockCreditRepo) UpdateScheduleItemStatus(ctx context.Context, itemID int64, status domain.PaymentStatus) error {
	return m.updateScheduleItemStatusFn(ctx, itemID, status)
}

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
	return m.updateBalanceFn(ctx, id, balance)
}

func TestCreateCredit(t *testing.T) {
	tests := []struct {
		name        string
		accountID   int64
		userID      int64
		amount      int64
		rate        float64
		term        int64
		setupMock   func(creditRepo *mockCreditRepo, accountRepo *mockAccountRepo)
		expectedErr error
	}{
		{
			name:      "успешное оформление кредита",
			accountID: 1,
			userID:    5,
			amount:    10000000,
			rate:      15.0,
			term:      12,
			setupMock: func(creditRepo *mockCreditRepo, accountRepo *mockAccountRepo) {
				accountRepo.findByIDFn = func(ctx context.Context, id int64) (*domain.Account, error) {
					return &domain.Account{ID: 1, UserID: 5, Balance: 100000}, nil
				}
				creditRepo.createCreditFn = func(ctx context.Context, credit *domain.Credit) error {
					credit.ID = 1
					return nil
				}
				creditRepo.createScheduleItemFn = func(ctx context.Context, item *domain.CreditScheduleItem) error {
					return nil
				}
			},
			expectedErr: nil,
		},
		{
			name:      "ошибка счет не найден",
			accountID: 999,
			userID:    5,
			amount:    10000000,
			rate:      15.0,
			term:      12,
			setupMock: func(creditRepo *mockCreditRepo, accountRepo *mockAccountRepo) {
				accountRepo.findByIDFn = func(ctx context.Context, id int64) (*domain.Account, error) {
					return nil, nil
				}
			},
			expectedErr: ErrCreditNotFound,
		},
		{
			name:      "ошибка: счет не принадлежит пользователю",
			accountID: 1,
			userID:    5,
			amount:    10000000,
			rate:      15.0,
			term:      12,
			setupMock: func(creditRepo *mockCreditRepo, accountRepo *mockAccountRepo) {
				accountRepo.findByIDFn = func(ctx context.Context, id int64) (*domain.Account, error) {
					return &domain.Account{ID: 1, UserID: 99}, nil
				}
			},
			expectedErr: ErrForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			creditRepo := &mockCreditRepo{}
			accountRepo := &mockAccountRepo{}
			tt.setupMock(creditRepo, accountRepo)

			svc := NewService(creditRepo, accountRepo)

			_, _, err := svc.CreateCredit(context.Background(), tt.accountID, tt.userID, tt.amount, tt.rate, tt.term)
			if tt.expectedErr != nil {
				require.ErrorIs(t, err, tt.expectedErr)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestGetSchedule(t *testing.T) {
	tests := []struct {
		name        string
		creditID    int64
		userID      int64
		setupMock   func(creditRepo *mockCreditRepo, accountREpo *mockAccountRepo)
		expectedLen int
		expectedErr error
	}{
		{
			name:     "успешное получение графика",
			creditID: 1,
			userID:   5,
			setupMock: func(creditRepo *mockCreditRepo, accountRepo *mockAccountRepo) {
				creditRepo.findByIDFn = func(ctx context.Context, id int64) (*domain.Credit, error) {
					return &domain.Credit{ID: 1, AccountID: 1}, nil
				}
				accountRepo.findByIDFn = func(ctx context.Context, id int64) (*domain.Account, error) {
					return &domain.Account{ID: 1, UserID: 5}, nil
				}
				creditRepo.findScheduleByCreditIDFn = func(ctx context.Context, creditID int64) ([]*domain.CreditScheduleItem, error) {
					return []*domain.CreditScheduleItem{
						{ID: 1, CreditID: 1},
						{ID: 2, CreditID: 1},
					}, nil
				}
			},
			expectedLen: 2,
			expectedErr: nil,
		},
		{
			name:     "ошибка: кредит не найден",
			creditID: 999,
			userID:   5,
			setupMock: func(creditRepo *mockCreditRepo, accountRepo *mockAccountRepo) {
				creditRepo.findByIDFn = func(ctx context.Context, id int64) (*domain.Credit, error) {
					return nil, nil
				}
			},
			expectedLen: 0,
			expectedErr: ErrCreditNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			creditRepo := &mockCreditRepo{}
			accountRepo := &mockAccountRepo{}
			tt.setupMock(creditRepo, accountRepo)

			svc := NewService(creditRepo, accountRepo)

			schedule, err := svc.GetSchedule(context.Background(), tt.creditID, tt.userID)

			if tt.expectedErr != nil {
				require.ErrorIs(t, err, tt.expectedErr)
				return
			}

			require.NoError(t, err)
			require.Len(t, schedule, tt.expectedLen)
		})
	}
}

func TestMakePayment(t *testing.T) {
	tests := []struct {
		name        string
		creditID    int64
		userID      int64
		setupMocks  func(creditRepo *mockCreditRepo, accountRepo *mockAccountRepo)
		expectedErr error
	}{
		{
			name:     "успешный платеж",
			creditID: 1,
			userID:   5,
			setupMocks: func(creditRepo *mockCreditRepo, accountRepo *mockAccountRepo) {
				creditRepo.findByIDFn = func(ctx context.Context, id int64) (*domain.Credit, error) {
					return &domain.Credit{ID: 1, AccountID: 1}, nil
				}
				accountRepo.findByIDFn = func(ctx context.Context, id int64) (*domain.Account, error) {
					return &domain.Account{ID: 1, UserID: 5, Balance: 100000}, nil
				}
				creditRepo.findScheduleByCreditIDFn = func(ctx context.Context, creditID int64) ([]*domain.CreditScheduleItem, error) {
					return []*domain.CreditScheduleItem{
						{ID: 1, CreditID: 1, Total: 5000, Status: domain.PaymentStatusPending},
					}, nil
				}
				accountRepo.updateBalanceFn = func(ctx context.Context, id int64, amount int64) error {
					return nil
				}
				creditRepo.updateScheduleItemStatusFn = func(ctx context.Context, itemID int64, status domain.PaymentStatus) error {
					return nil
				}
			},
			expectedErr: nil,
		},
		{
			name:     "ошибка: недостаточно средств",
			creditID: 1,
			userID:   5,
			setupMocks: func(creditRepo *mockCreditRepo, accountRepo *mockAccountRepo) {
				creditRepo.findByIDFn = func(ctx context.Context, id int64) (*domain.Credit, error) {
					return &domain.Credit{ID: 1, AccountID: 1}, nil
				}
				accountRepo.findByIDFn = func(ctx context.Context, id int64) (*domain.Account, error) {
					return &domain.Account{ID: 1, UserID: 5, Balance: 1000}, nil
				}
				creditRepo.findScheduleByCreditIDFn = func(ctx context.Context, id int64) ([]*domain.CreditScheduleItem, error) {
					return []*domain.CreditScheduleItem{
						{ID: 1, CreditID: 1, Total: 5000, Status: domain.PaymentStatusPending},
					}, nil
				}
			},
			expectedErr: ErrInsufficientFunds,
		},
		{
			name:     "ошибка: кредит не найден",
			creditID: 999,
			userID:   5,
			setupMocks: func(creditRepo *mockCreditRepo, accountRepo *mockAccountRepo) {
				creditRepo.findByIDFn = func(ctx context.Context, id int64) (*domain.Credit, error) {
					return nil, nil
				}
			},
			expectedErr: ErrCreditNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			creditRepo := &mockCreditRepo{}
			accountRepo := &mockAccountRepo{}
			tt.setupMocks(creditRepo, accountRepo)

			svc := NewService(creditRepo, accountRepo)

			err := svc.MakePayment(context.Background(), tt.creditID, tt.userID)
			if tt.expectedErr != nil {
				require.ErrorIs(t, err, tt.expectedErr)
				return
			}

			require.NoError(t, err)
		})
	}
}
