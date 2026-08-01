package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestNewCredit(t *testing.T) {
	tests := []struct {
		name        string
		accountUD   int64
		amount      int64
		rate        float64
		term        int64
		expectedErr error
	}{
		{
			name:        "успешное создание кредита",
			accountUD:   1,
			amount:      10000000,
			rate:        15.0,
			term:        12,
			expectedErr: nil,
		},
		{
			name:        "ошибка: сумма = 0",
			accountUD:   1,
			amount:      0,
			rate:        15.0,
			term:        12,
			expectedErr: ErrInvalidCreditAmount,
		},
		{
			name:        "ошибка: ставка = 0",
			accountUD:   1,
			amount:      10000000,
			rate:        0,
			term:        12,
			expectedErr: ErrInvalidCreditRate,
		},
		{
			name:        "ошибка: срок > 360 месяцев",
			accountUD:   1,
			amount:      10000000,
			rate:        15.0,
			term:        361,
			expectedErr: ErrInvalidCreditTerm,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			credit, err := NewCredit(tt.accountUD, tt.amount, tt.rate, tt.term)
			if tt.expectedErr != nil {
				require.ErrorIs(t, err, tt.expectedErr)
				require.Nil(t, credit)
				return
			}

			require.NoError(t, err)
			require.Equal(t, CreditStatusActive, credit.Status, "новый кредит должен быть active")
			require.Equal(t, tt.accountUD, credit.AccountID)
			require.Equal(t, tt.amount, credit.Amount)
			require.Equal(t, tt.rate, credit.Rate)
			require.Equal(t, tt.term, credit.TermMonths)
		})
	}
}

func TestCalculateAnnuityPayment(t *testing.T) {
	tests := []struct {
		name    string
		amount  int64
		rate    float64
		term    int64
		checkFn func(t *testing.T, payment int64, err error)
	}{
		{
			name:   "расчет для 100000 руб на 12 мес под 15%",
			amount: 10000000,
			rate:   15.0,
			term:   12,
			checkFn: func(t *testing.T, payment int64, err error) {
				require.NoError(t, err, "расчет не должен вернуть ошибку")
				require.Greater(t, payment, int64(0), "платеж должен быть положительным")
				require.Less(t, payment, int64(10000000), "платеж не может быть больше суммы кредита")
			},
		},
		{
			name:   "расчет для 50000 руб на 24 мес под 20%",
			amount: 50000000,
			rate:   20.0,
			term:   24,
			checkFn: func(t *testing.T, payment int64, err error) {
				require.NoError(t, err, "расчет не должен вернуть ошибку")
				require.Greater(t, payment, int64(0))
			},
		},
		{
			name:   "ошибка: неверная сумма",
			amount: 0,
			rate:   15.0,
			term:   12,
			checkFn: func(t *testing.T, payment int64, err error) {
				require.ErrorIs(t, err, ErrInvalidCreditAmount)
				require.Equal(t, int64(0), payment)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payment, err := calculateAnnuityPayment(tt.amount, tt.rate, tt.term)
			tt.checkFn(t, payment, err)
		})
	}
}

func TestGenerateSchedule(t *testing.T) {
	t.Run("график содержит правильное количество платежей", func(t *testing.T) {
		credit, err := NewCredit(1, 10000000, 15.0, 12)
		require.NoError(t, err)

		schedule := GenerateSchedule(credit, time.Now())

		require.Len(t, schedule, 12, "должно быть 12 платежей - по одному на каждый месяц")
	})

	t.Run("последний платеж обнуляет остаток", func(t *testing.T) {
		credit, err := NewCredit(1, 10000000, 15.0, 12)
		require.NoError(t, err)

		schedule := GenerateSchedule(credit, time.Now())
		lastPayment := schedule[len(schedule)-1]

		require.Equal(t, int64(0), lastPayment.RemainingBalance, "после последнего платежа остаток должен быть 0")
	})

	t.Run("каждый платеж имеет положительные principal и interest", func(t *testing.T) {
		credit, err := NewCredit(1, 10000000, 15.0, 12)
		require.NoError(t, err)

		schedule := GenerateSchedule(credit, time.Now())

		for i, payment := range schedule {
			require.Greater(t, payment.Principal, int64(0), "платеж %d: principal должен быть > 0", i)
			require.Greater(t, payment.Interest, int64(0), "платеж %d: interest должен быть > 0", i)
			require.Equal(t, PaymentStatusPending, payment.Status, "все платежи должны иметь статус pending (ожидают оплаты)")
		}
	})
}
