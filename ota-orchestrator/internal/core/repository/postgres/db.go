package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/Arondy/OTA-Firmware-Orchestrator/ota-orchestrator/internal/core/config"
	"github.com/Arondy/OTA-Firmware-Orchestrator/ota-orchestrator/internal/core/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

type executor interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

type DB struct {
	pool            *pgxpool.Pool
	tm              *TxManager
	requestTimeout  time.Duration
	paginationLimit int
}

func NewDB(ctx context.Context, config config.DBConfig, logger *zap.SugaredLogger) (*DB, error) {
	logger.Debugf("Connecting to Postgres on %s:%d", config.Host, config.Port)

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
		pool:            pool,
		tm:              NewTxManager(pool),
		requestTimeout:  config.RequestTimeout,
		paginationLimit: config.PaginationLimit,
	}, nil
}

func (d *DB) exec(ctx context.Context) executor {
	tx, ok := txFromContext(ctx)
	if ok {
		return tx
	}

	return d.pool
}

func (d *DB) TxManager() *TxManager {
	return d.tm
}

func (d *DB) addPagination(query string, args []any, pagination domain.Pagination) (string, []any) {
	if pagination.Limit == 0 || pagination.Limit > d.paginationLimit {
		pagination.Limit = d.paginationLimit
	}
	if pagination.Page == 0 {
		pagination.Page = 1
	}

	args = append(args, pagination.Limit)
	args = append(args, (pagination.Page-1)*pagination.Limit)

	query += fmt.Sprintf("\nLIMIT $%d OFFSET $%d\n", len(args)-1, len(args))
	return query, args
}

func (d *DB) Close() {
	d.pool.Close()
}
