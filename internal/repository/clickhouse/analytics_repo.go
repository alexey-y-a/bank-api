package clickhouse

import (
	"context"
	"fmt"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/alexey-y-a/bank-api/internal/config"
	"github.com/alexey-y-a/bank-api/internal/domain"
)

const insertTransactionsQuery = "INSERT INTO transactions_log (event_id, event_type, from_account_id, to_account_id, amount, created_at)"

type AnalyticsRepo struct {
	conn driver.Conn
}

func NewAnalyticsRepo(cfg config.ClickHouseConfig) (*AnalyticsRepo, error) {
	conn, err := clickhouse.Open(&clickhouse.Options{
		Addr: []string{cfg.Addr},
		Auth: clickhouse.Auth{
			Database: cfg.Database,
			Username: cfg.Username,
			Password: cfg.Password,
		},
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		return nil, fmt.Errorf("open clickhouse connection: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = conn.Ping(ctx)
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("ping clickhouse: %w", err)
	}

	return &AnalyticsRepo{conn: conn}, nil
}

func (r *AnalyticsRepo) SaveBatch(ctx context.Context, events []*domain.Event) error {
	if len(events) == 0 {
		return nil
	}

	batch, err := r.conn.PrepareBatch(ctx, insertTransactionsQuery)
	if err != nil {
		return fmt.Errorf("prepare clickhouse batch: %w", err)
	}

	for _, event := range events {
		err = batch.Append(
			event.ID,
			string(event.Type),
			event.FromAccountID,
			event.ToAccountID,
			event.Amount,
			event.CreatedAt.UTC(),
		)
		if err != nil {
			return fmt.Errorf("append event %q to clickhouse batch: %w", event.ID, err)
		}
	}

	err = batch.Send()
	if err != nil {
		return fmt.Errorf("send clickhouse batch: %w", err)
	}

	return nil
}

func (r *AnalyticsRepo) Close() error {
	return r.conn.Close()
}
