// Package database owns the PostgreSQL connection pool.
package database

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DB wraps the pgx pool so callers depend on this package rather than on pgx
// directly, and so the pool can grow transaction helpers later.
type DB struct {
	Pool *pgxpool.Pool
}

// New builds the pool from a connection URL.
//
// It deliberately does NOT ping: pgxpool connects lazily, and that laziness is
// what lets `go run ./cmd/api openapi` emit the spec without a live database.
// Readiness is checked by the health module at request time instead.
func New(ctx context.Context, url string) (*DB, error) {
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		return nil, fmt.Errorf("parse database url: %w", err)
	}

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("create connection pool: %w", err)
	}
	return &DB{Pool: pool}, nil
}

// Close releases every pooled connection.
func (db *DB) Close() {
	db.Pool.Close()
}

// InTx runs fn inside a transaction, committing on success and rolling back on
// any error or panic. Callers that must write several tables atomically — such
// as provisioning a user together with their oauth account — use this.
func (db *DB) InTx(ctx context.Context, fn func(tx pgx.Tx) error) error {
	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	// Rollback after commit is a no-op, so this is safe on the success path too.
	defer func() { _ = tx.Rollback(ctx) }()

	if err := fn(tx); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}
	return nil
}
