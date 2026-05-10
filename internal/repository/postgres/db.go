package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/alexey-y-a/bank-api/internal/config"
	"github.com/jackc/pgx/v5/pgxpool"
)

type DB interface {
	Close()
	Ping(ctx context.Context) error
	Pool() *pgxpool.Pool
}
type db struct {
	pool *pgxpool.Pool
}

func NewDB(cfg config.DatabaseConfig) (DB, error) {
	poolConfig, err := pgxpool.ParseConfig(cfg.DSN)
	if err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	poolConfig.MaxConns = int32(cfg.MaxOpenConns)

	poolConfig.MaxConns = int32(cfg.MaxIdleConns)

	poolConfig.MaxConnLifetime = time.Duration(cfg.ConnMaxLifetime) * time.Second

	pool, err := pgxpool.NewWithConfig(context.Background(), poolConfig)
	if err != nil {
		return nil, fmt.Errorf("create pool: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = pool.Ping(ctx)
	if err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping pool: %w", err)
	}

	return &db{pool: pool}, nil
}

func (d *db) Close() {
	if d.pool != nil {
		d.pool.Close()
	}
}

func (d *db) Ping(ctx context.Context) error {
	return d.pool.Ping(ctx)
}

func (d *db) Pool() *pgxpool.Pool {
	return d.pool
}
