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

const insertTransactionsQuery = "INSERT INTO transactions_log (event_id, event_type, from_account_id, to_account_id, amount, created_at)"

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

const incomeExpenseQuery = `
SELECT 
	sumIf(amount, has(?, to_account_id)) AS income,
	sumIf(amount, has(?, from_account_id)) AS expense
FROM transactions_log
WHERE created_at >= ? AND created_at < ?
`

func (r *AnalyticsRepo) IncomeExpense(ctx context.Context, accountIDs []int64, from, to time.Time) (domain.PeriodSummary, error) {
	var summary domain.PeriodSummary

	row := r.conn.QueryRow(ctx, incomeExpenseQuery, accountIDs, accountIDs, from, to)

	err := row.Scan(&summary.Income, &summary.Expense)
	if err != nil {
		return domain.PeriodSummary{}, fmt.Errorf("query clickhouse income/expense: %w", err)
	}

	return summary, nil
}

const topOperationsQuery = `
SELECT event_id, event_type, from_account_id, to_account_id, amount, created_at
FROM transactions_log
WHERE (has(?, from_account_id) OR has(?, to_account_id))
	AND created_at >= ? 
	AND created_at < ?
ORDER BY amount DESC 
LIMIT ?
`

func (r *AnalyticsRepo) TopOperations(ctx context.Context, accountIDs []int64, from, to time.Time, limit int) ([]*domain.Event, error) {
	rows, err := r.conn.Query(ctx, topOperationsQuery, accountIDs, accountIDs, from, to, limit)
	if err != nil {
		return nil, fmt.Errorf("query clickhouse top operations: %w", err)
	}

	defer func() {
		_ = rows.Close()
	}()

	var events []*domain.Event

	for rows.Next() {
		var event domain.Event
		var eventType string

		err = rows.Scan(
			&event.ID,
			&eventType,
			&event.FromAccountID,
			&event.ToAccountID,
			&event.Amount,
			&event.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan clickhouse top operation: %w", err)
		}

		event.Type = domain.EventType(eventType)
		events = append(events, &event)
	}

	err = rows.Err()
	if err != nil {
		return nil, fmt.Errorf("iterate clickhouse top operations: %w", err)
	}

	return events, nil
}

func (r *AnalyticsRepo) Close() error {
	return r.conn.Close()
}
