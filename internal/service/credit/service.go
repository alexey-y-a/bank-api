package credit

import (
	"context"
	"fmt"
	"time"

	"github.com/alexey-y-a/bank-api/internal/domain"
	"github.com/alexey-y-a/bank-api/internal/repository"
)

type Service struct {
	creditRepo  repository.CreditRepository
	accountRepo repository.AccountRepository
}

func NewService(creditREpo repository.CreditRepository, accountRepo repository.AccountRepository) *Service {
	return &Service{
		creditRepo:  creditREpo,
		accountRepo: accountRepo,
	}
}

func (s *Service) CreateCredit(ctx context.Context, accountID, userID int64, amount int64, rate float64, term int64) (*domain.Credit, []*domain.CreditScheduleItem, error) {
	account, err := s.accountRepo.FindByID(ctx, accountID)
	if err != nil {
		return nil, nil, fmt.Errorf("credit_service.CreateCredit find account: %w", err)
	}

	if account == nil {
		return nil, nil, ErrCreditNotFound
	}

	if account.UserID != userID {
		return nil, nil, ErrForbidden
	}

	credit, err := domain.NewCredit(accountID, amount, rate, term)
	if err != nil {
		return nil, nil, fmt.Errorf("credit_service.CreateCredit new credit: %w", err)
	}

	err = s.creditRepo.CreateCredit(ctx, credit)
	if err != nil {
		return nil, nil, fmt.Errorf("credit_service.CreateCredit save in db: %w", err)
	}

	schedule := domain.GenerateSchedule(credit, time.Now().AddDate(0, 1, 0))
	if schedule == nil {
		return nil, nil, fmt.Errorf("credit_service.CreateCredit generate schedule")
	}

	for _, item := range schedule {
		err = s.creditRepo.CreateScheduleItem(ctx, item)
		if err != nil {
			return nil, nil, fmt.Errorf("credit_service.CreateCredit save schedule in db: %w", err)
		}
	}

	return credit, schedule, nil
}

func (s *Service) GetSchedule(ctx context.Context, creditID, userID int64) ([]*domain.CreditScheduleItem, error) {
	credit, err := s.creditRepo.FindByID(ctx, creditID)
	if err != nil {
		return nil, fmt.Errorf("credit_service.GetSchedule find credit: %w", err)
	}

	if credit == nil {
		return nil, ErrCreditNotFound
	}

	account, err := s.accountRepo.FindByID(ctx, credit.AccountID)
	if err != nil {
		return nil, fmt.Errorf("credit_service.GetSchedule find account: %w", err)
	}

	if account == nil || account.UserID != userID {
		return nil, ErrForbidden
	}

	schedule, err := s.creditRepo.FindScheduleByCreditID(ctx, creditID)
	if err != nil {
		return nil, fmt.Errorf("credit_service.GetSchedule find schedule by credit id: %w", err)
	}

	return schedule, nil
}

func (s *Service) MakePayment(ctx context.Context, creditID, userID int64) error {
	credit, err := s.creditRepo.FindByID(ctx, creditID)
	if err != nil {
		return fmt.Errorf("credit_service.MakePayment find credit: %w", err)
	}

	if credit == nil {
		return ErrCreditNotFound
	}

	account, err := s.accountRepo.FindByID(ctx, credit.AccountID)
	if err != nil {
		return fmt.Errorf("credit_service.MakePayment find account: %w", err)
	}

	if account == nil || account.UserID != userID {
		return ErrForbidden
	}

	schedule, err := s.creditRepo.FindScheduleByCreditID(ctx, creditID)
	if err != nil {
		return fmt.Errorf("credit_service.MakePayment find schedule by credit id: %w", err)
	}

	var nextPayment *domain.CreditScheduleItem
	for _, item := range schedule {
		if item.Status == domain.PaymentStatusPending {
			nextPayment = item
			break
		}
	}

	if nextPayment == nil {
		return fmt.Errorf("credit_service.MakePayment: no pending payments")
	}

	err = account.Withdraw(nextPayment.Total)
	if err != nil {
		return ErrInsufficientFunds
	}

	err = s.accountRepo.UpdateBalance(ctx, account.ID, account.Balance)
	if err != nil {
		return fmt.Errorf("credit_service.MakePayment update balance: %w", err)
	}

	err = s.creditRepo.UpdateScheduleItemStatus(ctx, nextPayment.ID, domain.PaymentStatusPaid)
	if err != nil {
		return fmt.Errorf("credit_service.MakePayment update status: %w", err)
	}

	return nil
}
