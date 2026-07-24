package domain

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewTransaction(t *testing.T) {
	tests := []struct {
		name           string
		fromAccountID  int64
		toAccountID    int64
		amount         int64
		txType         TransactionType
		description    string
		expectedErr    error
		expectedStatus TransactionStatus
	}{
		{
			name:           "успешное создание перевода",
			fromAccountID:  1,
			toAccountID:    2,
			amount:         5000,
			txType:         TransactionTypeTransfer,
			description:    "перевод между счетами",
			expectedErr:    nil,
			expectedStatus: TransactionStatusCompleted,
		},
		{
			name:           "успешное создание пополнения",
			fromAccountID:  0,
			toAccountID:    1,
			amount:         10000,
			txType:         TransactionTypeDeposit,
			description:    "пополнение с карты",
			expectedErr:    nil,
			expectedStatus: TransactionStatusCompleted,
		},
		{
			name:        "ошибка сумма 0",
			amount:      0,
			txType:      TransactionTypeDeposit,
			expectedErr: ErrInvalidAmount,
		},
		{
			name:        "ошибка отрицательная сумма",
			amount:      -500,
			txType:      TransactionTypeDeposit,
			expectedErr: ErrInvalidAmount,
		},
		{
			name:        "ошибка неверный тип операции",
			amount:      1000,
			txType:      "invalid",
			expectedErr: ErrInvalidTransactionType,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tx, err := NewTransaction(tt.fromAccountID, tt.toAccountID, tt.amount, tt.txType, tt.description)
			if tt.expectedErr != nil {
				require.ErrorIs(t, err, tt.expectedErr, "должна быть ошибка %v", tt.expectedErr)
				require.Nil(t, tx, "при ошибке транзакция должна быть nil")
				return
			}

			require.NoError(t, err, "не должно быть ошибки")
			require.NotZero(t, tx.CreatedAt, "время создания должно быть заполнено")
			require.Equal(t, tt.expectedStatus, tx.Status, "статус должен быть %s", tt.expectedStatus)
			require.Equal(t, tt.amount, tx.Amount, "сумма должна совпадать")
			require.Equal(t, tt.txType, tx.Type, "тип должен совпадать")
		})
	}
}
