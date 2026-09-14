package database

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// Migrate applies any pending embedded goose migrations. It is safe to run
// on every startup: goose tracks applied versions in goose_db_version.
func Migrate(ctx context.Context, pool *pgxpool.Pool, logger *slog.Logger) error {
	fsys, err := fs.Sub(migrationsFS, "migrations")
	if err != nil {
		return fmt.Errorf("migrations fs: %w", err)
	}

	// goose speaks database/sql; this adapter borrows connections from the
	// pgx pool. Closing it does not close the pool.
	db := stdlib.OpenDBFromPool(pool)
	defer db.Close()

	provider, err := goose.NewProvider(goose.DialectPostgres, db, fsys)
	if err != nil {
		return fmt.Errorf("goose provider: %w", err)
	}

	results, err := provider.Up(ctx)
	if err != nil {
		return fmt.Errorf("apply migrations: %w", err)
	}
	for _, r := range results {
		logger.Info("migration applied",
			"version", r.Source.Version,
			"file", r.Source.Path,
			"duration", r.Duration,
		)
	}
	if len(results) == 0 {
		logger.Info("migrations up to date")
	}
	return nil
}
