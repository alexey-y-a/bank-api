package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/alexey-y-a/bank-api/internal/domain"
	"github.com/alexey-y-a/bank-api/internal/test"
	"github.com/stretchr/testify/require"
)

func TestUserRepository_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()

	dbHelper, err := test.NewPostgresHelper(ctx)
	require.NoError(t, err, "failed to start postgres container")

	defer func() {
		_ = dbHelper.Cleanup()
	}()

	err = dbHelper.RunMigrations("migrations")
	require.NoError(t, err, "failed to run migrations")

	repo := NewUserRepository(dbHelper.Pool)

	t.Run("Create", func(t *testing.T) {
		user, err := domain.NewUser("test@example.com", "ivan_ivanov", "hashed_secret_123")
		require.NoError(t, err)

		err = repo.Create(ctx, user)
		require.NoError(t, err, "create user failed")

		require.NotZero(t, user.ID, "ID must be populated after create")

		require.WithinDuration(t, time.Now(), user.CreatedAt, 2*time.Second)
	})

	t.Run("FindByEmail", func(t *testing.T) {
		found, err := repo.FindByEmail(ctx, "test@example.com")
		require.NoError(t, err, "find by email failed")

		require.NotNil(t, found, "user must be found")

		require.Equal(t, "ivan_ivanov", found.Username)
		require.Equal(t, "test@example.com", found.Email)
	})

	t.Run("FindByEmailNotFound", func(t *testing.T) {
		found, err := repo.FindByEmail(ctx, "nonexistent@example.com")
		require.NoError(t, err, "find by email should not return error for not found")

		require.Nil(t, found, "user should be nil when not found")
	})

	t.Run("FindByID", func(t *testing.T) {
		user, err := domain.NewUser("test2@example.com", "petr_petrov", "pass_hash")
		require.NoError(t, err)

		err = repo.Create(ctx, user)
		require.NoError(t, err)

		found, err := repo.FindByID(ctx, user.ID)
		require.NoError(t, err)
		require.NotNil(t, found)

		require.Equal(t, user.ID, found.ID)
		require.Equal(t, "test2@example.com", found.Email)
	})

	t.Run("Create_DuplicateEmail", func(t *testing.T) {
		user1, err := domain.NewUser("unique@example.com", "user1", "pass123456")
		require.NoError(t, err)

		err = repo.Create(ctx, user1)
		require.NoError(t, err)

		user2, err := domain.NewUser("unique@example.com", "user2", "pass1234567")
		require.NoError(t, err)

		err = repo.Create(ctx, user2)
		require.Error(t, err, "creating duplicate email should fail")
	})
}
