package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/Arondy/OTA-Firmware-Orchestrator/internal/core/config"
	"github.com/jackc/pgx/v5/pgxpool"
)

type DB struct {
	pool           *pgxpool.Pool
	requestTimeout time.Duration
}

func NewDB(ctx context.Context, config config.DBConfig) (*DB, error) {
	pgxConfig, err := pgxpool.ParseConfig(config.ConnString())
	if err != nil {
		return nil, fmt.Errorf("failed to parse connString: %w", err)
	}

	pool, err := pgxpool.NewWithConfig(ctx, pgxConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create pgxpool: %w", err)
	}

	if err = pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed ping: %w", err)
	}

	return &DB{
		pool:           pool,
		requestTimeout: config.RequestTimeout,
	}, nil
}

func (r *DB) Close() {
	r.pool.Close()
}
