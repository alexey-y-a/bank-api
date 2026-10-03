package scheduler

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alexey-y-a/bank-api/internal/domain"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

type mockCreditRepo struct {
	findPendingPaymentsBeforeFn func(ctx context.Context, beforeDate time.Time) ([]*domain.CreditScheduleItem, error)
	updatePaymentPenaltyFn      func(ctx context.Context, itemID, penalty int64) error
	updateScheduleItemStatusFn  func(ctx context.Context, itemID int64, status domain.PaymentStatus) error
	updateStatusFn              func(ctx context.Context, id int64, status domain.CreditStatus) error
}

func (m *mockCreditRepo) CreateCredit(ctx context.Context, credit *domain.Credit) error {
	return nil
}

func (m *mockCreditRepo) FindByID(ctx context.Context, id int64) (*domain.Credit, error) {
	return nil, nil
}

func (m *mockCreditRepo) FindByAccountID(ctx context.Context, id int64) ([]*domain.Credit, error) {
	return nil, nil
}

func (m *mockCreditRepo) UpdateStatus(ctx context.Context, id int64, status domain.CreditStatus) error {
	if m.updateStatusFn != nil {
		return m.updateStatusFn(ctx, id, status)
	}
	return nil
}

func (m *mockCreditRepo) CreateScheduleItem(ctx context.Context, item *domain.CreditScheduleItem) error {
	return nil
}

func (m *mockCreditRepo) FindScheduleByCreditID(ctx context.Context, creditID int64) ([]*domain.CreditScheduleItem, error) {
	return nil, nil
}

func (m *mockCreditRepo) UpdateScheduleItemStatus(ctx context.Context, itemID int64, status domain.PaymentStatus) error {
	if m.updateScheduleItemStatusFn != nil {
		return m.updateScheduleItemStatusFn(ctx, itemID, status)
	}
	return nil
}

func (m *mockCreditRepo) FindPendingPaymentsBefore(ctx context.Context, beforeDate time.Time) ([]*domain.CreditScheduleItem, error) {
	return m.findPendingPaymentsBeforeFn(ctx, beforeDate)
}

func (m *mockCreditRepo) UpdatePaymentPenalty(ctx context.Context, itemID, penalty int64) error {
	if m.updatePaymentPenaltyFn != nil {
		return m.updatePaymentPenaltyFn(ctx, itemID, penalty)
	}
	return nil
}

func TestProcessPayment(t *testing.T) {
	log := logrus.New()
	log.SetOutput(discard{})

	payment := &domain.CreditScheduleItem{
		ID:       1,
		CreditID: 1,
		Total:    10000,
		Status:   domain.PaymentStatusPending,
	}

	penaltyUpdated := false

	repo := &mockCreditRepo{
		updatePaymentPenaltyFn: func(ctx context.Context, itemID int64, penalty int64) error {
			penaltyUpdated = true
			require.Equal(t, int64(1000), penalty, "пеня должна быть 10% от суммы")
			return nil
		},
		updateScheduleItemStatusFn: func(ctx context.Context, itemID int64, status domain.PaymentStatus) error {
			require.Equal(t, domain.PaymentStatusOverdue, status, "статус должен быть просрочен")
			return nil
		},
	}

	svc := NewScheduler(repo, log)

	err := svc.processPayment(context.Background(), payment)
	require.NoError(t, err, "не должно быть ошибок")
	require.True(t, penaltyUpdated, "пеня должна быть обновлена")
}

func TestProcessOverduePayments(t *testing.T) {
	log := logrus.New()
	log.SetOutput(discard{})

	t.Run("нет просроченных платежей", func(t *testing.T) {
		repo := &mockCreditRepo{
			findPendingPaymentsBeforeFn: func(ctx context.Context, beforeDate time.Time) ([]*domain.CreditScheduleItem, error) {
				return []*domain.CreditScheduleItem{}, nil
			},
		}

		svc := NewScheduler(repo, log)
		svc.processOverduePayments(context.Background())
	})

	t.Run("один просроченный платеж", func(t *testing.T) {
		creditStatusUpdated := false

		repo := &mockCreditRepo{
			findPendingPaymentsBeforeFn: func(ctx context.Context, beforeDate time.Time) ([]*domain.CreditScheduleItem, error) {
				return []*domain.CreditScheduleItem{
					{ID: 1, CreditID: 1, Total: 10000, Status: domain.PaymentStatusPending},
				}, nil
			},
			updatePaymentPenaltyFn: func(ctx context.Context, itemID int64, penalty int64) error {
				return nil
			},
			updateScheduleItemStatusFn: func(ctx context.Context, itemID int64, status domain.PaymentStatus) error {
				return nil
			},
			updateStatusFn: func(ctx context.Context, id int64, status domain.CreditStatus) error {
				creditStatusUpdated = true
				require.Equal(t, domain.CreditStatusOverdue, status, "статус должен стать просрочен")
				return nil
			},
		}

		svc := NewScheduler(repo, log)
		svc.processOverduePayments(context.Background())
		require.True(t, creditStatusUpdated, "статус кредита должен обновится")
	})

	t.Run("Ошибка при поиске платежей", func(t *testing.T) {
		repo := &mockCreditRepo{
			findPendingPaymentsBeforeFn: func(ctx context.Context, beforeDate time.Time) ([]*domain.CreditScheduleItem, error) {
				return nil, errors.New("db error")
			},
		}

		svc := NewScheduler(repo, log)
		svc.processOverduePayments(context.Background())
	})
}

type discard struct{}

func (discard) Write(p []byte) (int, error) {
	return len(p), nil
}
