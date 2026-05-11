package test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

type PostgresHelper struct {
	Container *postgres.PostgresContainer
	Pool      *pgxpool.Pool
	ctx       context.Context
	cancel    context.CancelFunc
}

func NewPostgresHelper(ctx context.Context) (*PostgresHelper, error) {
	testCtx, cancel := context.WithTimeout(ctx, 60*time.Second)

	pgContainer, err := postgres.Run(
		testCtx,
		"postgres:17-alpine",
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("testuser"),
		postgres.WithPassword("testpass"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).WithStartupTimeout(30*time.Second),
		),
	)

	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to run postgres container: %w", err)
	}

	connStr, err := pgContainer.ConnectionString(testCtx, "sslmode=disable")
	if err != nil {
		cancel()
		_ = pgContainer.Terminate(testCtx)
		return nil, fmt.Errorf("failed to get database connection string: %w", err)
	}

	pool, err := pgxpool.New(testCtx, connStr)
	if err != nil {
		cancel()
		_ = pgContainer.Terminate(testCtx)
		return nil, fmt.Errorf("failed to create pgxpool: %w", err)
	}

	return &PostgresHelper{
		Container: pgContainer,
		Pool:      pool,
		ctx:       testCtx,
		cancel:    cancel,
	}, nil
}

func (h *PostgresHelper) RunMigrations(migrationsDir string) error {
	if !filepath.IsAbs(migrationsDir) {
		root, err := findProjectRoot()
		if err != nil {
			return fmt.Errorf("failed to find project root: %w", err)
		}
		migrationsDir = filepath.Join(root, migrationsDir)
	}

	files, err := os.ReadDir(migrationsDir)
	if err != nil {
		return fmt.Errorf("failed to read migrations dir: %w", err)
	}

	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".up.sql") {
			continue
		}

		path := filepath.Join(migrationsDir, file.Name())
		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read migrations %s: %w", file.Name(), err)
		}

		_, err = h.Pool.Exec(h.ctx, string(content))
		if err != nil {
			return fmt.Errorf("failed to execute migration %s: %w", file.Name(), err)
		}
	}

	return nil
}

func (h *PostgresHelper) Cleanup() error {
	if h.Pool != nil {
		h.Pool.Close()
	}

	if h.cancel != nil {
		h.cancel()
	}

	terminateCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if h.Container != nil {
		return h.Container.Terminate(terminateCtx)
	}

	return nil
}

func findProjectRoot() (string, error) {

	dir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get working directory: %w", err)
	}

	for {
		_, err := os.Stat(filepath.Join(dir, "go.mod"))
		if err == nil {

			return dir, nil
		}

		parent := filepath.Dir(dir)

		if parent == dir {
			return "", fmt.Errorf("go.mod not found in any parent directory")
		}

		dir = parent
	}
}
