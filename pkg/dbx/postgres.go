package dbx

import (
	"context"
	"fmt"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	_defaultMaxPoolSize  = 1
	_defaultConnAttempts = 10
	_defaultConnTimeout  = time.Second
)

// Postgres pgxpool wrapper
type Postgres struct {
	maxPoolSize  int32
	connAttempts int
	connTimeout  time.Duration

	Builder squirrel.StatementBuilderType
	Pool    *pgxpool.Pool
}

// NewPostgres postgres connection pool
func NewPostgres(url string, opts ...Option) (*Postgres, error) {
	pg := &Postgres{
		maxPoolSize:  _defaultMaxPoolSize,
		connAttempts: _defaultConnAttempts,
		connTimeout:  _defaultConnTimeout,
	}

	// Custom options
	for _, opt := range opts {
		opt(pg)
	}

	pg.Builder = squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar)

	poolConfig, err := pgxpool.ParseConfig(url)
	if err != nil {
		return nil, fmt.Errorf("postgres - NewPostgres - pgxpool.ParseConfig: %w", err)
	}

	poolConfig.MaxConns = pg.maxPoolSize

	for pg.connAttempts > 0 {
		pg.Pool, err = pgxpool.NewWithConfig(context.Background(), poolConfig)
		if err == nil {
			break
		}

		time.Sleep(pg.connTimeout)

		pg.connAttempts--
	}

	if err != nil {
		return nil, fmt.Errorf("postgres - NewPostgres - connAttempts == 0: %w", err)
	}

	return pg, nil
}

// Select -.
func (p *Postgres) Select(ctx context.Context, dest any, builder squirrel.SelectBuilder) error {
	if p.Pool == nil {
		return fmt.Errorf("postgres - Select - p.Pool == nil")
	}

	sql, args, err := builder.ToSql()
	if err != nil {
		return fmt.Errorf("postgres - Select - builder.ToSql: %w", err)
	}

	if err := pgxscan.Select(ctx, p.Pool, &dest, sql, args...); err != nil {
		return fmt.Errorf("postgres - Select - pgxscan.Select: %w", err)
	}

	return nil
}

// Get -.
func (p *Postgres) Get(ctx context.Context, dest any, builder squirrel.SelectBuilder) error {
	if p.Pool == nil {
		return fmt.Errorf("postgres - Get - p.Pool == nil")
	}

	sql, args, err := builder.ToSql()
	if err != nil {
		return fmt.Errorf("postgres - Get - builder.ToSql: %w", err)
	}

	if err := pgxscan.Get(ctx, p.Pool, &dest, sql, args...); err != nil {
		return fmt.Errorf("postgres - Get - pgxscan.Get: %w", err)
	}

	return nil
}

// QueryRow -.
func (p *Postgres) ScanOne(ctx context.Context, dest any, builder squirrel.SelectBuilder) error {
	if p.Pool == nil {
		return fmt.Errorf("postgres - QueryRow - p.Pool == nil")
	}

	sql, args, err := builder.ToSql()
	if err != nil {
		return fmt.Errorf("postgres - QueryRow - builder.ToSql: %w", err)
	}

	rows, err := p.Pool.Query(ctx, sql, args...)
	if err != nil {
		return fmt.Errorf("postgres - QueryRow - p.Pool.Query: %w", err)
	}
	defer rows.Close()

	if err := pgxscan.ScanOne(dest, rows); err != nil {
		return fmt.Errorf("postgres - QueryRow - pgxscan.ScanOne: %w", err)
	}

	return nil
}

// ScanAll -.
func (p *Postgres) ScanAll(ctx context.Context, dest any, builder squirrel.SelectBuilder) error {
	if p.Pool == nil {
		return fmt.Errorf("postgres - ScanAll - p.Pool == nil")
	}

	sql, args, err := builder.ToSql()
	if err != nil {
		return fmt.Errorf("postgres - ScanAll - builder.ToSql: %w", err)
	}

	rows, err := p.Pool.Query(ctx, sql, args...)
	if err != nil {
		return fmt.Errorf("postgres - ScanAll - p.Pool.Query: %w", err)
	}
	defer rows.Close()

	if err := pgxscan.ScanAll(dest, rows); err != nil {
		return fmt.Errorf("postgres - ScanAll - pgxscan.ScanAll: %w", err)
	}

	return nil
}

func (p *Postgres) GetTx(ctx context.Context) (pgx.Tx, error) {
	if p.Pool == nil {
		return nil, fmt.Errorf("postgres - GetTx - p.Pool == nil")
	}

	tx, err := p.Pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("postgres - GetTx - p.Pool.Begin: %w", err)
	}

	return tx, nil
}

// Close the connection pool
func (p *Postgres) Close() {
	if p.Pool != nil {
		p.Pool.Close()
	}
}
