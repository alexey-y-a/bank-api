package domain

import (
	"errors"
	"time"
)

var (
	ErrInvalidAmount     = errors.New("amount must be greater than zero")
	ErrInsufficientFunds = errors.New("insufficient funds")
)

type Account struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"user_id"`
	Balance   int64     `json:"balance"`
	Currency  string    `json:"currency"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func NewAccount(userID int64, currency string) *Account {
	now := time.Now()

	return &Account{
		UserID:    userID,
		Balance:   0,
		Currency:  currency,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func (a *Account) Deposit(amount int64) error {
	if amount <= 0 {
		return ErrInvalidAmount
	}

	a.Balance += amount

	a.UpdatedAt = time.Now()

	return nil
}

func (a *Account) Withdraw(amount int64) error {
	if amount <= 0 {
		return ErrInvalidAmount
	}

	if a.Balance < amount {
		return ErrInsufficientFunds
	}

	a.Balance -= amount
	a.UpdatedAt = time.Now()

	return nil
}

func (a *Account) CanWithdraw(amount int64) bool {
	return a.Balance >= amount
}
