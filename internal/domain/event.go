package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

type EventType string

const (
	EventTypeTransfer   EventType = "transfer"
	EventTypeDeposit    EventType = "deposit"
	EventTypeWithdrawal EventType = "withdrawal"
	EventTypePayment    EventType = "payment"
	EventTypeCredit     EventType = "credit"
)

var (
	ErrInvalidEventType   = errors.New("invalid event type")
	ErrInvalidEventAmount = errors.New("event amount must be greater than zero")
)

type Event struct {
	ID            string    `json:"id"`
	Type          EventType `json:"type"`
	FromAccountID int64     `json:"from_account_id"`
	ToAccountID   int64     `json:"to_account_id"`
	Amount        int64     `json:"amount"`
	CreatedAt     time.Time `json:"created_at"`
}

func NewEvent(eventType EventType, fromAccountID, toAccountID, amount int64) (*Event, error) {
	switch eventType {
	case EventTypeTransfer, EventTypeDeposit, EventTypeWithdrawal, EventTypePayment, EventTypeCredit:
	default:
		return nil, ErrInvalidEventType
	}

	if amount <= 0 {
		return nil, ErrInvalidEventAmount
	}

	return &Event{
		ID:            uuid.NewString(),
		Type:          eventType,
		FromAccountID: fromAccountID,
		ToAccountID:   toAccountID,
		Amount:        amount,
		CreatedAt:     time.Now(),
	}, nil
}
