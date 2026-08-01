package domain

import (
	"errors"
	"math"
	"time"
)

var (
	ErrInvalidCreditAmount = errors.New("credit amount must be greater than zero")
	ErrInvalidCreditRate   = errors.New("interest rate must be greater than zero")
	ErrInvalidCreditTerm   = errors.New("term must be between 1 and 360 months")
)

type CreditStatus string

const (
	CreditStatusActive  CreditStatus = "active"
	CreditStatusClosed  CreditStatus = "closed"
	CreditStatusOverdue CreditStatus = "overdue"
)

type PaymentStatus string

const (
	PaymentStatusPending PaymentStatus = "pending"
	PaymentStatusPaid    PaymentStatus = "paid"
	PaymentStatusOverdue PaymentStatus = "overdue"
)

type Credit struct {
	ID         int64        `json:"id"`
	AccountID  int64        `json:"account_id"`
	Amount     int64        `json:"amount"`
	Rate       float64      `json:"rate"`
	TermMonths int64        `json:"term_months"`
	Status     CreditStatus `json:"status"`
	CreatedAt  time.Time    `json:"created_at"`
	UpdatedAt  time.Time    `json:"updated_at"`
}

type CreditScheduleItem struct {
	ID               int64         `json:"id"`
	CreditID         int64         `json:"credit_id"`
	PaymentDate      time.Time     `json:"payment_date"`
	Principal        int64         `json:"principal"`
	Interest         int64         `json:"interest"`
	Total            int64         `json:"total"`
	RemainingBalance int64         `json:"remaining_balance"`
	Status           PaymentStatus `json:"status"`
}

func NewCredit(accountID, amount int64, rate float64, term int64) (*Credit, error) {
	if amount <= 0 {
		return nil, ErrInvalidCreditAmount
	}

	if rate <= 0 {
		return nil, ErrInvalidCreditRate
	}

	if term < 1 || term > 360 {
		return nil, ErrInvalidCreditTerm
	}

	now := time.Now()

	return &Credit{
		AccountID:  accountID,
		Amount:     amount,
		Rate:       rate,
		TermMonths: term,
		Status:     CreditStatusActive,
		CreatedAt:  now,
		UpdatedAt:  now,
	}, nil
}

func calculateAnnuityPayment(amount int64, rate float64, term int64) (int64, error) {
	if amount <= 0 {
		return 0, ErrInvalidCreditAmount
	}

	if rate <= 0 {
		return 0, ErrInvalidCreditRate
	}

	if term < 1 || term > 360 {
		return 0, ErrInvalidCreditTerm
	}

	monthyRate := rate / 12.0 / 100.0

	pow := math.Pow(1+monthyRate, float64(term))

	coefficient := (monthyRate * pow) / (pow - 1)

	payment := float64(amount) * coefficient

	return int64(math.Ceil(payment)), nil
}

func GenerateSchedule(credit *Credit, startDate time.Time) []*CreditScheduleItem {
	payment, err := calculateAnnuityPayment(credit.Amount, credit.Rate, credit.TermMonths)
	if err != nil {
		return nil
	}

	monthlyRate := credit.Rate / 12.0 / 100.0

	remaining := credit.Amount

	schedule := make([]*CreditScheduleItem, credit.TermMonths)

	term := int(credit.TermMonths)

	for i := 0; i < term; i++ {
		interest := int64(math.Floor(float64(remaining) * monthlyRate))

		principal := payment - interest

		remaining -= principal

		if remaining < 0 {
			principal += remaining
			remaining = 0
		}

		schedule[i] = &CreditScheduleItem{
			CreditID:         credit.ID,
			PaymentDate:      startDate.AddDate(0, i, 0),
			Principal:        principal,
			Interest:         interest,
			Total:            principal + interest,
			RemainingBalance: remaining,
			Status:           PaymentStatusPending,
		}
	}

	return schedule
}
