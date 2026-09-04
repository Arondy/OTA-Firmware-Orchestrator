//go:build integration

package postgres

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
)

var (
	testDB      *DB
	pgContainer *tcpostgres.PostgresContainer
)

// terminateAndFatal stops the already-started container before exiting, so a
// failed startup does not leak it (the testcontainers reaper is best-effort).
func terminateAndFatal(format string, args ...any) {
	if pgContainer != nil {
		_ = pgContainer.Terminate(context.Background())
	}
	log.Fatalf(format, args...)
}

func TestMain(m *testing.M) {
	ctx := context.Background()

	container, err := tcpostgres.Run(ctx,
		"postgres:18-alpine",
		tcpostgres.WithDatabase("testdb"),
		tcpostgres.WithUsername("test"),
		tcpostgres.WithPassword("test"),
		tcpostgres.BasicWaitStrategies(),
	)
	if err != nil {
		log.Fatalf("failed to start postgres container: %v", err)
	}
	pgContainer = container

	connStr, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		terminateAndFatal("failed to get connection string: %v", err)
	}

	pgxConfig, err := pgxpool.ParseConfig(connStr)
	if err != nil {
		terminateAndFatal("failed to parse pool config: %v", err)
	}
	pgxConfig.MaxConns = 10

	pool, err := pgxpool.NewWithConfig(ctx, pgxConfig)
	if err != nil {
		terminateAndFatal("failed to create pool: %v", err)
	}

	if err := applyMigrations(context.Background(), pool); err != nil {
		terminateAndFatal("failed to apply migrations: %v", err)
	}

	testDB = &DB{pool: pool, tm: NewTxManager(pool), requestTimeout: 10 * time.Second}

	code := m.Run()

	testDB.Close()
	_ = container.Terminate(ctx)
	os.Exit(code)
}

func applyMigrations(ctx context.Context, pool *pgxpool.Pool) error {
	// The schema relies on a uuidv7() function; provide a self-contained
	// implementation so the tests are runnable without the pg_uuidv7 extension.
	if _, err := pool.Exec(ctx, `CREATE OR REPLACE FUNCTION uuidv7() RETURNS uuid LANGUAGE sql AS $$ SELECT gen_random_uuid(); $$;`); err != nil {
		return fmt.Errorf("failed to create uuidv7 function: %w", err)
	}

	dir := migrationsDirPath()
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("failed to read migrations dir: %w", err)
	}

	var upFiles []string
	for _, e := range entries {
		if matched, _ := filepath.Match("*.up.sql", e.Name()); matched {
			upFiles = append(upFiles, e.Name())
		}
	}
	sort.Strings(upFiles)

	for _, name := range upFiles {
		content, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return fmt.Errorf("failed to read migration %s: %w", name, err)
		}
		if _, err := pool.Exec(ctx, string(content)); err != nil {
			return fmt.Errorf("migration %s failed: %w", name, err)
		}
	}
	return nil
}

func migrationsDirPath() string {
	// file is in ota-orchestrator/internal/core/repository/postgres;
	// repo root (with migrations/) is 5 levels up.
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		panic("failed to resolve current file path")
	}
	dir := filepath.Join(filepath.Dir(thisFile), "..", "..", "..", "..", "..", "migrations")
	abs, err := filepath.Abs(dir)
	if err != nil {
		panic(err)
	}
	return abs
}

func resetDB(t *testing.T) {
	t.Helper()
	ctx := context.Background()
	_, err := testDB.pool.Exec(ctx, `
		TRUNCATE applied_decisions, update_attempts, rollout_stages, rollout_campaigns, firmware_versions, devices RESTART IDENTITY CASCADE;
	`)
	require.NoError(t, err)
}
