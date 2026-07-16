package domain

import (
	"errors"
	"time"
)

var (
	ErrInvalidCardNumber = errors.New("invalid card number: luhn check failed")
	ErrCardExpired       = errors.New("card has expired")
	ErrCardBlocked       = errors.New("card is blocked")
)

type CardStatus string

const (
	CardStatusActive  CardStatus = "active"
	CardStatusBlocked CardStatus = "blocked"
	CardStatusExpired CardStatus = "expired"
)

type Card struct {
	ID        int64      `json:"id"`
	AccountID int64      `json:"account_id"`
	Number    string     `json:"number"`
	CVV       string     `json:"-"`
	ExpiresAt time.Time  `json:"expires_at"`
	Status    CardStatus `json:"status"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

func NewCard(accountID int64) (*Card, error) {
	number := GenerateCardNumber()

	if !ValidLuhn(number) {
		return nil, ErrInvalidCardNumber
	}

	now := time.Now()

	return &Card{
		AccountID: accountID,
		Number:    number,
		ExpiresAt: time.Date(now.Year()+3, now.Month(), 1, 0, 0, 0, 0, time.UTC),
		Status:    CardStatusActive,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

func (c *Card) MaskNumber() string {
	if len(c.Number) < 4 {
		return "**** **** **** ****"
	}

	last4 := c.Number[len(c.Number)-4:]

	return "**** **** **** " + last4
}

func (c *Card) CanUse() error {
	if c.Status == CardStatusBlocked {
		return ErrCardBlocked
	}

	if time.Now().After(c.ExpiresAt) {
		return ErrCardExpired
	}

	return nil
}
