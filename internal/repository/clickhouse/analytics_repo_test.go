package clickhouse

import (
	"context"
	"testing"

	"github.com/alexey-y-a/bank-api/internal/config"
	"github.com/alexey-y-a/bank-api/internal/domain"
	"github.com/stretchr/testify/require"
)

func TestSaveBatchEmptyBatch(t *testing.T) {
	repo := &AnalyticsRepo{}

	err := repo.SaveBatch(context.Background(), nil)
	require.NoError(t, err)
}

func TestSaveBatchIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test requires ClickHouse from docker-compose")
	}

	repo, err := NewAnalyticsRepo(config.ClickHouseConfig{
		Addr:     "localhost:9000",
		Database: "analytics",
		Username: "bank",
		Password: "bank",
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, repo.Close())
	})

	first, err := domain.NewEvent(domain.EventTypeTransfer, 1, 2, 2500)
	require.NoError(t, err)

	second, err := domain.NewEvent(domain.EventTypeDeposit, 0, 5, 1000)
	require.NoError(t, err)

	err = repo.SaveBatch(context.Background(), []*domain.Event{first, second})
	require.NoError(t, err)

	var count uint64

	err = repo.conn.QueryRow(
		context.Background(), "SELECT count() FROM transactions_log WHERE event_id IN (?, ?)",
		first.ID, second.ID,
	).Scan(&count)
	require.NoError(t, err)
	require.Equal(t, uint64(2), count)
}
