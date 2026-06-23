package domain

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewAccount(t *testing.T) {
	tests := []struct {
		name     string
		userID   int64
		currency string
	}{
		{
			name:     "счет в рублях для пользователя 1",
			userID:   1,
			currency: "RUB",
		},
		{
			name:     "счет в долларах для пользователя 2",
			userID:   2,
			currency: "USD",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			acc := NewAccount(tt.userID, tt.currency)

			require.Equal(t, tt.userID, acc.UserID, "владелец счета должен совпадать")
			require.Equal(t, tt.currency, acc.Currency, "Валюта должна совпадать")

			require.EqualValues(t, 0, acc.Balance, "у нового счета баланс 0")

			require.NotZero(t, acc.CreatedAt, "время создания должно быть заполнено")
			require.NotZero(t, acc.UpdatedAt, "время обновления должно быть заполнено")
		})
	}
}

func TestAccount_Deposit(t *testing.T) {
	acc := NewAccount(1, "RUB")

	tests := []struct {
		name          string
		amount        int64
		expectedErr   error
		expectedAfter int64
	}{
		{
			name:          "пополнение на 1000 - успех",
			amount:        1000,
			expectedErr:   nil,
			expectedAfter: 1000,
		},
		{
			name:          "пополнение на 0",
			amount:        0,
			expectedErr:   ErrInvalidAmount,
			expectedAfter: 1000,
		},
		{
			name:          "пополнение на отрицательную сумму - ошибка",
			amount:        -500,
			expectedErr:   ErrInvalidAmount,
			expectedAfter: 1000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := acc.Deposit(tt.amount)

			if tt.expectedErr != nil {
				require.ErrorIs(t, err, tt.expectedErr, "должна быть ошибка %v", tt.expectedErr)
			} else {
				require.NoError(t, err, "ошибки не должно быть")
			}

			require.Equal(t, tt.expectedAfter, acc.Balance, "баланс должен %d", tt.expectedAfter)
		})
	}
}

func TestAccount_Withdraw(t *testing.T) {
	acc := NewAccount(1, "RUB")

	err := acc.Deposit(5000)
	require.NoError(t, err, "должны положить 5000 перед тестом списания")

	tests := []struct {
		name          string
		amount        int64
		expectedErr   error
		expectedAfter int64
	}{
		{
			name:          "списание 2000 - успех",
			amount:        2000,
			expectedErr:   nil,
			expectedAfter: 3000,
		},
		{
			name:          "списание больше баланса - ошибка",
			amount:        5000,
			expectedErr:   ErrInsufficientFunds,
			expectedAfter: 3000,
		},
		{
			name:          "списание 0 - ошибка",
			amount:        0,
			expectedErr:   ErrInvalidAmount,
			expectedAfter: 3000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := acc.Withdraw(tt.amount)

			if tt.expectedErr != nil {
				require.ErrorIs(t, err, tt.expectedErr)
			} else {
				require.NoError(t, err)
			}

			require.Equal(t, tt.expectedAfter, acc.Balance, ",")
		})
	}
}

func TestAccount_CanWithdraw(t *testing.T) {
	acc := NewAccount(1, "RUB")

	err := acc.Deposit(3000)
	require.NoError(t, err)

	tests := []struct {
		name   string
		amount int64
		want   bool
	}{
		{
			name:   "3000 можно снять 1000",
			amount: 1000,
			want:   true,
		},
		{
			name:   "3000 можно снять 3000",
			amount: 3000,
			want:   true,
		},
		{
			name:   "3000 нельзя снять 3100",
			amount: 3100,
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := acc.CanWithdraw(tt.amount)

			require.Equal(t, tt.want, got, "CanWithdraw(%d) должен вернуть %v", tt.amount, tt.want)
		})
	}
}
