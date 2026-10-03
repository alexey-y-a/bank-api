package scheduler

import (
	"context"
	"math"
	"sync"
	"time"

	"github.com/alexey-y-a/bank-api/internal/domain"
	"github.com/alexey-y-a/bank-api/internal/repository"
	"github.com/sirupsen/logrus"
)

const (
	checkInterval  = 12 * time.Hour
	penaltyPercent = 10.0
)

type Scheduler struct {
	creditRepo repository.CreditRepository
	log        *logrus.Logger
	wg         sync.WaitGroup
}

func NewScheduler(creditRepo repository.CreditRepository, log *logrus.Logger) *Scheduler {
	return &Scheduler{
		creditRepo: creditRepo,
		log:        log,
	}
}

func (s *Scheduler) Run(ctx context.Context) {
	s.wg.Go(func() {
		s.log.WithFields(logrus.Fields{
			"interval": checkInterval.String(),
		}).Info("credit scheduler started")

		ticker := time.NewTicker(checkInterval)
		defer ticker.Stop()

		s.processOverduePayments(ctx)

		for {
			select {
			case <-ctx.Done():
				s.log.Info("credit scheduler stopped")
				return

			case <-ticker.C:
				s.processOverduePayments(ctx)
			}
		}
	})
}

func (s *Scheduler) Stop() {
	s.wg.Wait()
}

func (s *Scheduler) processOverduePayments(ctx context.Context) {
	now := time.Now()

	payments, err := s.creditRepo.FindPendingPaymentsBefore(ctx, now)
	if err != nil {
		s.log.WithError(err).Error("scheduler: failed to find overdue payments")
		return
	}

	if len(payments) == 0 {
		s.log.Debug("scheduler: no overdue payments found")
		return
	}

	s.log.WithFields(logrus.Fields{
		"count": len(payments),
	}).Info("scheduler: processing overdue payments")

	updatedCredits := make(map[int64]bool)

	for i := range payments {
		payment := payments[i]

		err := s.processPayment(ctx, payment)
		if err != nil {
			s.log.WithError(err).WithFields(logrus.Fields{
				"payment_id": payment.ID,
				"credit_id":  payment.CreditID,
			}).Error("scheduler: failed to process payment")
			continue
		}

		alreadyUpdated := updatedCredits[payment.CreditID]
		if !alreadyUpdated {
			err := s.creditRepo.UpdateStatus(ctx, payment.CreditID, domain.CreditStatusOverdue)
			if err != nil {
				s.log.WithError(err).WithFields(logrus.Fields{
					"credit_id": payment.CreditID,
				}).Error("scheduler: failed to update status")
			}
			updatedCredits[payment.CreditID] = true
		}
	}
}

func (s *Scheduler) processPayment(ctx context.Context, payment *domain.CreditScheduleItem) error {
	penaltyFloat := float64(payment.Total) * penaltyPercent / 100.0
	penalty := int64(math.Round(penaltyFloat))

	err := s.creditRepo.UpdatePaymentPenalty(ctx, payment.ID, penalty)
	if err != nil {
		return err
	}

	err = s.creditRepo.UpdateScheduleItemStatus(ctx, payment.ID, domain.PaymentStatusOverdue)
	if err != nil {
		return err
	}

	s.log.WithFields(logrus.Fields{
		"payment_id": payment.ID,
		"credit_id":  payment.CreditID,
		"total":      payment.Total,
		"penalty":    penalty,
	}).Info("scheduler: payment marked as overdue")

	return nil
}
