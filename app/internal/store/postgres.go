package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Postgres is a Store backed by a PostgreSQL connection pool.
type Postgres struct {
	pool *pgxpool.Pool
}

func NewPostgres(ctx context.Context, dsn string) (*Postgres, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("open pool: %w", err)
	}
	p := &Postgres{pool: pool}
	if err := p.migrate(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return p, nil
}

// migrate creates the table if needed. One table, so a simple CREATE is enough.
func (p *Postgres) migrate(ctx context.Context) error {
	_, err := p.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS users (
			username      TEXT PRIMARY KEY,
			date_of_birth DATE NOT NULL
		)`)
	return err
}

func (p *Postgres) Upsert(ctx context.Context, username string, dob time.Time) error {
	_, err := p.pool.Exec(ctx, `
		INSERT INTO users (username, date_of_birth)
		VALUES ($1, $2)
		ON CONFLICT (username) DO UPDATE SET date_of_birth = EXCLUDED.date_of_birth`,
		username, dob)
	return err
}

func (p *Postgres) Get(ctx context.Context, username string) (time.Time, bool, error) {
	var dob time.Time
	err := p.pool.QueryRow(ctx,
		`SELECT date_of_birth FROM users WHERE username = $1`, username).Scan(&dob)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return time.Time{}, false, nil
		}
		return time.Time{}, false, err
	}
	return dob, true, nil
}

func (p *Postgres) Ping(ctx context.Context) error { return p.pool.Ping(ctx) }

func (p *Postgres) Close() error {
	p.pool.Close()
	return nil
}
