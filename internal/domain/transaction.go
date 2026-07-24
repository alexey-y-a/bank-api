package domain

import (
	"errors"
	"time"
)

type TransactionType string

const (
	TransactionTypeTransfer   TransactionType = "transfer"
	TransactionTypeDeposit    TransactionType = "deposit"
	TransactionTypeWithdrawal TransactionType = "withdrawal"
	TransactionTypePayment    TransactionType = "payment"
)

type TransactionStatus string

const (
	TransactionStatusPending   TransactionStatus = "pending"
	TransactionStatusCompleted TransactionStatus = "completed"
	TransactionStatusFailed    TransactionStatus = "failed"
)

var (
	ErrInvalidTransactionType   = errors.New("invalid transaction type")
	ErrInvalidTransactionStatus = errors.New("invalid transaction status")
)

type Transaction struct {
	ID            int64             `json:"id"`
	FromAccountID int64             `json:"from_account_id"`
	ToAccountID   int64             `json:"to_account_id"`
	Amount        int64             `json:"amount"`
	Type          TransactionType   `json:"type"`
	Status        TransactionStatus `json:"status"`
	Description   string            `json:"description"`
	CreatedAt     time.Time         `json:"created_at"`
}

func NewTransaction(fromAccountID, toAccountID int64, amount int64, txType TransactionType, description string) (*Transaction, error) {
	if amount <= 0 {
		return nil, ErrInvalidAmount
	}

	switch txType {
	case TransactionTypeTransfer, TransactionTypeDeposit, TransactionTypeWithdrawal, TransactionTypePayment:
	default:
		return nil, ErrInvalidTransactionType
	}

	return &Transaction{
		FromAccountID: fromAccountID,
		ToAccountID:   toAccountID,
		Amount:        amount,
		Type:          txType,
		Status:        TransactionStatusCompleted,
		Description:   description,
		CreatedAt:     time.Now(),
	}, nil
}
