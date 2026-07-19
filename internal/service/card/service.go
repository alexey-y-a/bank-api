package card

import (
	"context"
	"fmt"

	"github.com/alexey-y-a/bank-api/internal/crypto"
	"github.com/alexey-y-a/bank-api/internal/domain"
	"github.com/alexey-y-a/bank-api/internal/repository"
)

type Service struct {
	encryptor   *crypto.CardEncryptor
	cardRepo    repository.CardRepository
	accountRepo repository.AccountRepository
}

func NewService(encryptor *crypto.CardEncryptor, cardRepo repository.CardRepository, accountRepo repository.AccountRepository) *Service {
	return &Service{
		encryptor:   encryptor,
		cardRepo:    cardRepo,
		accountRepo: accountRepo,
	}
}

func (s *Service) IssueCard(ctx context.Context, accountID, userID int64) (*domain.Card, error) {
	account, err := s.accountRepo.FindByID(ctx, accountID)
	if err != nil {
		return nil, fmt.Errorf("card_service.IssueCard find account: %w", err)
	}

	if account == nil {
		return nil, fmt.Errorf("card_service.IssueCard: %w", ErrCardNotFound)
	}

	if account.UserID != userID {
		return nil, fmt.Errorf("card_service.IssueCard: %w", ErrCardNotFound)
	}

	card, err := domain.NewCard(accountID)
	if err != nil {
		return nil, fmt.Errorf("card_service.IssueCard new card: %w", err)
	}

	card.CVV = "123"

	numberHash, numberEnc, _, err := s.encryptor.EncryptNumber(card.Number)
	if err != nil {
		return nil, fmt.Errorf("card_service.IssueCard encrypt number: %w", err)
	}

	cvvHash, err := s.encryptor.HashCVV(card.CVV)
	if err != nil {
		return nil, fmt.Errorf("card_service.IssueCard hash cvv: %w", err)
	}

	card.Number = numberHash

	err = s.cardRepo.Create(ctx, card)
	if err != nil {
		return nil, fmt.Errorf("card_service.IssueCard create: %w", err)
	}

	card.Number = string(numberEnc)
	card.CVV = cvvHash

	return card, nil
}

func (s *Service) GetUserCards(ctx context.Context, userID int64) ([]*domain.Card, error) {
	accounts, err := s.accountRepo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("card_service.GetUserCards find accounts: %w", err)
	}

	var allCards []*domain.Card

	for _, account := range accounts {
		cards, err := s.cardRepo.FindByAccountID(ctx, account.ID)
		if err != nil {
			return nil, fmt.Errorf("card_service.GetUserCards find cards: %w", err)
		}

		for _, c := range cards {
			number, err := s.encryptor.DecryptNumber([]byte(c.Number))
			if err != nil {
				continue
			}
			c.Number = number
		}

		allCards = append(allCards, cards...)
	}

	if allCards == nil {
		return []*domain.Card{}, nil
	}

	return allCards, nil
}

func (s *Service) BlockCard(ctx context.Context, cardID, userID int64) error {
	card, err := s.cardRepo.FindByID(ctx, cardID)
	if err != nil {
		return fmt.Errorf("card_service.BlockCard find: %w", err)
	}

	if card == nil {
		return ErrCardNotFound
	}

	account, err := s.accountRepo.FindByID(ctx, card.AccountID)
	if err != nil {
		return fmt.Errorf("card_service.BlockCard find account: %w", err)
	}

	if account == nil {
		return ErrCardNotFound
	}

	if account.UserID != userID {
		return ErrCardNotFound
	}

	err = s.cardRepo.UpdateStatus(ctx, cardID, domain.CardStatusBlocked)
	if err != nil {
		return fmt.Errorf("card_service.BlockCard update: %w", err)
	}

	return nil
}

func (s *Service) PayWithCard(ctx context.Context, cardID, userID int64, cvv string, amount int64) error {
	card, err := s.cardRepo.FindByID(ctx, cardID)
	if err != nil {
		return fmt.Errorf("card_service.PayWithCard find: %w", err)
	}

	if card == nil {
		return ErrCardNotFound
	}

	err = s.encryptor.VerifyCVV(card.CVV, cvv)
	if err != nil {
		return ErrInvalidCVV
	}

	err = card.CanUse()
	if err != nil {
		return domain.ErrCardBlocked
	}

	account, err := s.accountRepo.FindByID(ctx, card.AccountID)
	if err != nil {
		return fmt.Errorf("card_service.PayWithCard find account: %w", err)
	}

	if account == nil {
		return ErrCardNotFound
	}

	if account.UserID != userID {
		return ErrCardNotFound
	}

	err = account.Withdraw(amount)
	if err != nil {
		return fmt.Errorf("card_service.PayWithCard withdraw: %w", err)
	}

	err = s.accountRepo.UpdateBalance(ctx, account.ID, account.Balance)
	if err != nil {
		return fmt.Errorf("card_service.PayWithCard update balance: %w", err)
	}

	return nil
}
