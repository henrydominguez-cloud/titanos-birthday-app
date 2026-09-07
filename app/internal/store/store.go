// Package store defines how the app persists users and ships two backends:
// an in-memory one (tests) and a PostgreSQL one (default).
package store

import (
	"context"
	"time"
)

// Store is the persistence interface the API depends on.
type Store interface {
	Upsert(ctx context.Context, username string, dob time.Time) error
	Get(ctx context.Context, username string) (dob time.Time, found bool, err error)
	Ping(ctx context.Context) error // used by the readiness probe
	Close() error
}
