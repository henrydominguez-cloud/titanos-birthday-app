// Command server starts the birthday HTTP API.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/titanos/birthday-app/internal/api"
	"github.com/titanos/birthday-app/internal/store"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	if err := run(logger); err != nil {
		logger.Error("fatal", "err", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	port := getenv("PORT", "8080")
	backend := getenv("STORE_BACKEND", "postgres")

	// Cancel the context on Ctrl+C / SIGTERM so we can shut down cleanly.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	st, err := newStore(ctx, backend, logger)
	if err != nil {
		return err
	}
	defer st.Close()

	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           api.NewServer(st, logger).Routes(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	errc := make(chan error, 1)
	go func() {
		logger.Info("server listening", "addr", srv.Addr, "backend", backend)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errc <- err
		}
	}()

	select {
	case err := <-errc:
		return err
	case <-ctx.Done():
		logger.Info("shutting down")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		return srv.Shutdown(shutdownCtx)
	}
}

// newStore picks the backend. For postgres it retries the first connection so
// the app can start before the database pod is ready.
func newStore(ctx context.Context, backend string, logger *slog.Logger) (store.Store, error) {
	switch backend {
	case "memory":
		return store.NewMemory(), nil
	case "postgres":
		dsn := os.Getenv("DATABASE_URL")
		if dsn == "" {
			return nil, errors.New("DATABASE_URL is required when STORE_BACKEND=postgres")
		}
		deadline := time.Now().Add(60 * time.Second)
		for {
			pg, err := store.NewPostgres(ctx, dsn)
			if err == nil {
				return pg, nil
			}
			if time.Now().After(deadline) {
				return nil, err
			}
			logger.Warn("database not ready, retrying", "err", err)
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(2 * time.Second):
			}
		}
	default:
		return nil, errors.New("unknown STORE_BACKEND: " + backend)
	}
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
