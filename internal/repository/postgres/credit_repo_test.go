package postgres

import (
	"testing"
	"time"

	"github.com/alexey-y-a/bank-api/internal/domain"
	"github.com/alexey-y-a/bank-api/internal/repository"
	"github.com/alexey-y-a/bank-api/internal/test"
	"github.com/stretchr/testify/require"
)

func TestCreditRepository_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := t.Context()

	dbHelper, err := test.NewPostgresHelper(ctx)
	require.NoError(t, err, "должен запуститься PostgreSQL контейнер")

	err = dbHelper.RunMigrations("migrations")
	require.NoError(t, err, "должны выполнить миграции")

	creditRepo := NewCreditRepo(dbHelper.Pool)

	userRepo := NewUserRepository(dbHelper.Pool)
	accountRepo := NewAccountRepository(dbHelper.Pool)

	user := createTestUser(t, userRepo)
	account := createTestAccount(t, accountRepo, user.ID)

	t.Cleanup(func() {
		_ = dbHelper.Cleanup()
	})

	t.Run("CreateCredit", func(t *testing.T) {
		credit, err := domain.NewCredit(account.ID, 10000000, 15.0, 12)
		require.NoError(t, err)

		err = creditRepo.CreateCredit(ctx, credit)
		require.NoError(t, err, "создание кредита не должно вернуть ошибку")

		require.NotZero(t, credit.ID, "ID кредита должен быть заполнен")
		require.Equal(t, account.ID, credit.AccountID, "счет должен совпадать")
		require.Equal(t, domain.CreditStatusActive, credit.Status, "статус должен быть active")
	})

	t.Run("FindByID", func(t *testing.T) {
		credit, err := domain.NewCredit(account.ID, 50000000, 20.0, 24)
		require.NoError(t, err)
		err = creditRepo.CreateCredit(ctx, credit)
		require.NoError(t, err)

		found, err := creditRepo.FindByID(ctx, credit.ID)
		require.NoError(t, err, "поиск не должен вернуть ошибку")
		require.NotNil(t, found, "кредит должен быть найден")
		require.Equal(t, credit.ID, found.ID, "ID должны совпадать")
		require.Equal(t, credit.Amount, found.Amount, "сумма должна совпадать")
		require.Equal(t, credit.Rate, found.Rate, "ставка должна совпадать")
	})

	t.Run("FundById_NotFound", func(t *testing.T) {
		found, err := creditRepo.FindByID(ctx, -1)
		require.NoError(t, err, "отсутствие записи не ошибка")
		require.Nil(t, found, "если записи нет, должен быть nil")
	})

	t.Run("FindByAccountID", func(t *testing.T) {
		otherAccount := createTestAccount(t, accountRepo, user.ID)

		credit1, err := domain.NewCredit(otherAccount.ID, 1000000, 10.0, 6)
		require.NoError(t, err)
		err = creditRepo.CreateCredit(ctx, credit1)
		require.NoError(t, err)

		credit2, err := domain.NewCredit(otherAccount.ID, 2000000, 12.0, 12)
		require.NoError(t, err)
		err = creditRepo.CreateCredit(ctx, credit2)
		require.NoError(t, err)

		// Ищем кредиты ТОЛЬКО этого счёта — их будет ровно 2
		credits, err := creditRepo.FindByAccountID(ctx, otherAccount.ID)
		require.NoError(t, err, "поиск не должен вернуть ошибку")
		require.Len(t, credits, 2, "должно быть 2 кредита")
	})

	t.Run("UpdateStatus", func(t *testing.T) {
		credit, err := domain.NewCredit(account.ID, 30000000, 15.0, 12)
		require.NoError(t, err)
		err = creditRepo.CreateCredit(ctx, credit)
		require.NoError(t, err)

		err = creditRepo.UpdateStatus(ctx, credit.ID, domain.CreditStatusClosed)
		require.NoError(t, err, "обновление статуса не должно вернуть ошибку")

		found, err := creditRepo.FindByID(ctx, credit.ID)
		require.NoError(t, err)
		require.Equal(t, domain.CreditStatusClosed, found.Status, "статус должен быть closed")
	})

	t.Run("Schedule", func(t *testing.T) {
		credit, err := domain.NewCredit(account.ID, 10000000, 15.0, 12)
		require.NoError(t, err)
		err = creditRepo.CreateCredit(ctx, credit)
		require.NoError(t, err)

		item := &domain.CreditScheduleItem{
			CreditID:         credit.ID,
			PaymentDate:      time.Now().AddDate(0, 1, 0),
			Principal:        50000,
			Interest:         15000,
			Total:            65000,
			RemainingBalance: 95000,
			Status:           domain.PaymentStatusPending,
		}

		err = creditRepo.CreateScheduleItem(ctx, item)
		require.NoError(t, err, "сохранение графика не должно вернуть ошибку")
		require.NotZero(t, item.ID, "ID элемента графика должен быть заполнен")

		items, err := creditRepo.FindScheduleByCreditID(ctx, credit.ID)
		require.NoError(t, err, "получение графика не должно вернуть ошибку")
		require.Len(t, items, 1, "должен быть 1 элемент графика")
		require.Equal(t, credit.ID, items[0].CreditID, "credit_id должен совпадать")
	})
}

func createTestAccount(t *testing.T, repo repository.AccountRepository, userID int64) *domain.Account {
	ctx := t.Context()

	account := domain.NewAccount(userID, "RUB")

	err := repo.Create(ctx, account)
	require.NoError(t, err)

	return account
}
