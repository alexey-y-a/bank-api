package postgres

import (
	"testing"
	"time"

	"github.com/alexey-y-a/bank-api/internal/domain"
	"github.com/alexey-y-a/bank-api/internal/repository"
	"github.com/alexey-y-a/bank-api/internal/test"
	"github.com/stretchr/testify/require"
)

func TestAccountRepository_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := t.Context()

	dbHelper, err := test.NewPostgresHelper(ctx)
	require.NoError(t, err, "должны запустить PostgreSQL контейнер")

	err = dbHelper.RunMigrations("migrations")
	require.NoError(t, err, "должны выполнить миграции")

	repo := NewAccountRepository(dbHelper.Pool)

	userRepo := NewUserRepository(dbHelper.Pool)

	user := createTestUser(t, userRepo)

	t.Cleanup(func() {
		_ = dbHelper.Cleanup()
	})

	t.Run("Create", func(t *testing.T) {
		account := domain.NewAccount(user.ID, "RUB")
		err := repo.Create(ctx, account)

		require.NoError(t, err, "создание счета не должно возвращать ошибку")
		require.NotZero(t, account.ID, "ID должен быть заполнен из БД")
		require.Equal(t, user.ID, account.UserID, "владелец счета должен совпадать")
		require.EqualValues(t, 0, account.Balance, "у нового счета баланс 0")
		require.Equal(t, "RUB", account.Currency, "валюта должна быть RUB")
		require.NotZero(t, account.CreatedAt, "время создания должно быть заполнено")
		require.NotZero(t, account.UpdatedAt, "время обновления должно быть заполнено")
	})

	t.Run("FindByID", func(t *testing.T) {
		account := domain.NewAccount(user.ID, "RUB")
		err := repo.Create(ctx, account)
		require.NoError(t, err)

		found, err := repo.FindByID(ctx, account.ID)

		require.NoError(t, err, "поиск по ID не должен возвращать ошибку")
		require.NotNil(t, found, "счет должен быть найден")
		require.Equal(t, account.ID, found.ID, "ID должен совпадать")
		require.Equal(t, account.UserID, found.UserID, "владелец должен совпадать")
		require.Equal(t, account.Balance, found.Balance, "баланс должен совпадать")
		require.Equal(t, account.Currency, found.Currency, "валюта должна совпадать")
	})

	t.Run("FindByID_NotFound", func(t *testing.T) {
		found, err := repo.FindByID(ctx, -1)

		require.NoError(t, err, "отсутствует запись - не ошибка")
		require.Nil(t, found, "если записи нет, должен быть nil")
	})

	t.Run("FindByUserID", func(t *testing.T) {
		otherUser := createTestUser(t, userRepo)

		acc1 := domain.NewAccount(otherUser.ID, "RUB")
		err := repo.Create(ctx, acc1)
		require.NoError(t, err)

		acc2 := domain.NewAccount(otherUser.ID, "RUB")
		err = repo.Create(ctx, acc2)
		require.NoError(t, err)

		accounts, err := repo.FindByUserID(ctx, otherUser.ID)

		require.NoError(t, err, "поиск по user_id не должен вернуть ошибку")
		require.Len(t, accounts, 2, "должно быть 2 счёта")
		require.Equal(t, acc2.ID, accounts[0].ID, "первый счёт в DESC — это acc2 (создан позже)")
		require.Equal(t, acc1.ID, accounts[1].ID, "второй счёт в DESC — это acc1 (создан раньше)")
	})

	t.Run("UpdateBalance", func(t *testing.T) {
		account := domain.NewAccount(user.ID, "RUB")
		err := repo.Create(ctx, account)
		require.NoError(t, err)

		err = repo.UpdateBalance(ctx, account.ID, 5000)
		require.NoError(t, err, "обновление баланса не должно вернуть ошибку")

		updated, err := repo.FindByID(ctx, account.ID)
		require.NoError(t, err)
		require.Equal(t, int64(5000), updated.Balance, "баланс должен быть 5000")
	})

	t.Run("UpdateBalance_NotFound", func(t *testing.T) {
		err := repo.UpdateBalance(ctx, -1, 1000)

		require.ErrorIs(t, err, repository.ErrNotFound, "должна быть ошибка record not found")
	})
}

func createTestUser(t *testing.T, repo repository.UserRepository) *domain.User {
	ctx := t.Context()

	user, err := domain.NewUser(
		"test-account-"+time.Now().Format("150405.000000000")+"@example.com",
		"testuser-"+time.Now().Format("150405.000000000"),
		"password123",
	)
	require.NoError(t, err)

	err = repo.Create(ctx, user)
	require.NoError(t, err)

	return user
}
