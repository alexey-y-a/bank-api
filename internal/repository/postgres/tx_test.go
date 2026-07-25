package postgres

import (
	"context"
	"errors"
	"testing"

	"github.com/alexey-y-a/bank-api/internal/test"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

func setupTxTest(t *testing.T) (*pgxpool.Pool, func()) {
	t.Helper()

	ctx := t.Context()

	dbHelper, err := test.NewPostgresHelper(ctx)
	require.NoError(t, err, "должны запустить PostgreSQL контейнер")

	err = dbHelper.RunMigrations("migrations")
	require.NoError(t, err, "должны выполнить миграции")

	cleanup := func() {
		_ = dbHelper.Cleanup()
	}

	return dbHelper.Pool, cleanup
}

func createTestUserAndAccount(t *testing.T, pool *pgxpool.Pool, initialBalance int64) (userID int64, accountID int64) {
	t.Helper()

	ctx := context.Background()

	var uid int64
	err := pool.QueryRow(ctx, "INSERT INTO users (email, username, password_hash) VALUES ($1, $2, $3) RETURNING id",
		"tx-test-"+t.Name()+"@example.com", "txuser-"+t.Name(), "hash").Scan(&uid)
	require.NoError(t, err)

	var aid int64
	err = pool.QueryRow(ctx, "INSERT INTO accounts (user_id, balance, currency) VALUES ($1, $2, 'RUB') RETURNING id",
		uid, initialBalance).Scan(&aid)
	require.NoError(t, err)

	return uid, aid
}

func getAccountBalance(t *testing.T, pool *pgxpool.Pool, accountID int64) int64 {
	t.Helper()

	var balance int64
	err := pool.QueryRow(context.Background(), "SELECT balance FROM accounts WHERE id = $1", accountID).Scan(&balance)
	require.NoError(t, err)

	return balance
}

func TestWithTransaction(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	pool, cleanup := setupTxTest(t)
	defer cleanup()

	_, accountID := createTestUserAndAccount(t, pool, 10000)

	err := WithTransaction(context.Background(), pool, func(tx pgx.Tx) error {
		_, execErr := tx.Exec(context.Background(), "UPDATE accounts SET balance = balance + $1 WHERE id = $2", 5000, accountID)

		return execErr
	})

	require.NoError(t, err, "транзакция должна выполниться успешно")

	balance := getAccountBalance(t, pool, accountID)
	require.Equal(t, int64(15000), balance, "баланс должен увеличиться на 5000")
}

func TestWithTransaction_Error(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	pool, cleanup := setupTxTest(t)
	defer cleanup()

	_, accountID := createTestUserAndAccount(t, pool, 10000)

	testErr := errors.New("test error")
	err := WithTransaction(context.Background(), pool, func(tx pgx.Tx) error {
		_, execErr := tx.Exec(context.Background(), "UPDATE accounts SET balance = balance + $1 WHERE id = $2", 5000, accountID)

		if execErr != nil {
			return execErr
		}

		return testErr
	})

	require.Error(t, err, "должна быть ошибка")
	require.Contains(t, err.Error(), "test error", "текст ошибки должен сохраниться")

	balance := getAccountBalance(t, pool, accountID)
	require.Equal(t, int64(10000), balance, "баланс не должен измениться при ошибке")
}

func TestWithTransaction_Panic(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	pool, cleanup := setupTxTest(t)
	defer cleanup()

	_, accountID := createTestUserAndAccount(t, pool, 10000)

	require.Panics(t, func() {
		_ = WithTransaction(context.Background(), pool, func(tx pgx.Tx) error {
			_, execErr := tx.Exec(context.Background(), "UPDATE accounts SET balance = balance + $1 WHERE id = $2", 5000, accountID)
			if execErr != nil {
				return execErr
			}

			panic("test panic")
		})
	}, "должна быть паника")

	balance := getAccountBalance(t, pool, accountID)
	require.Equal(t, int64(10000), balance, "баланс не должен измениться при панике")
}

func TestWithTransaction_MultipleOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	pool, cleanup := setupTxTest(t)
	defer cleanup()

	_, accountID1 := createTestUserAndAccount(t, pool, 10000)

	ctx := context.Background()
	var accountID2 int64
	err := pool.QueryRow(
		ctx,
		"INSERT INTO accounts (user_id, balance, currency) VALUES (1, 5000, 'RUB') RETURNING id",
	).Scan(&accountID2)
	require.NoError(t, err)

	err = WithTransaction(ctx, pool, func(tx pgx.Tx) error {
		_, execErr := tx.Exec(
			ctx,
			"UPDATE accounts SET balance = balance - $1 WHERE id = $2",
			3000, accountID1,
		)
		if execErr != nil {
			return execErr
		}

		_, execErr = tx.Exec(
			ctx,
			"UPDATE accounts SET balance = balance + $1 WHERE id = $2",
			3000, accountID2,
		)
		if execErr != nil {
			return execErr
		}

		return nil
	})

	require.NoError(t, err, "перевод должен выполниться успешно")

	balance1 := getAccountBalance(t, pool, accountID1)
	balance2 := getAccountBalance(t, pool, accountID2)

	require.Equal(t, int64(7000), balance1, "баланс первого счёта должен уменьшиться на 3000")
	require.Equal(t, int64(8000), balance2, "баланс второго счёта должен увеличиться на 3000")
}

func TestWithTransaction_MultipleOperationsRollback(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	pool, cleanup := setupTxTest(t)
	defer cleanup()

	_, accountID1 := createTestUserAndAccount(t, pool, 10000)

	ctx := context.Background()
	var accountID2 int64
	err := pool.QueryRow(
		ctx,
		"INSERT INTO accounts (user_id, balance, currency) VALUES (1, 5000, 'RUB') RETURNING id",
	).Scan(&accountID2)
	require.NoError(t, err)

	err = WithTransaction(ctx, pool, func(tx pgx.Tx) error {
		_, execErr := tx.Exec(
			ctx,
			"UPDATE accounts SET balance = balance - $1 WHERE id = $2",
			3000, accountID1,
		)
		if execErr != nil {
			return execErr
		}

		_, execErr = tx.Exec(
			ctx,
			"UPDATE accounts SET balance = balance + $1 WHERE id = $2",
			3000, accountID2,
		)
		if execErr != nil {
			return execErr
		}

		return errors.New("simulated error after both operations")
	})

	require.Error(t, err, "должна быть ошибка")

	balance1 := getAccountBalance(t, pool, accountID1)
	balance2 := getAccountBalance(t, pool, accountID2)

	require.Equal(t, int64(10000), balance1, "баланс первого счёта не должен измениться")
	require.Equal(t, int64(5000), balance2, "баланс второго счёта не должен измениться")
}
