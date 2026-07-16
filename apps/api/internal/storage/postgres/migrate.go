package postgres

import (
	"context"
	"embed"
	"fmt"
)

//go:embed migrations/*.up.sql
var migrationFiles embed.FS

func (r *Repository) Migrate(ctx context.Context) error {
	if _, err := r.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version bigint PRIMARY KEY,
			applied_at timestamptz NOT NULL DEFAULT now()
		)`); err != nil {
		return fmt.Errorf("ensure schema migrations table: %w", err)
	}
	var applied bool
	if err := r.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version = 1)`).Scan(&applied); err != nil {
		return fmt.Errorf("read migration version: %w", err)
	}
	if applied {
		return nil
	}
	script, err := migrationFiles.ReadFile("migrations/000001_foundation.up.sql")
	if err != nil {
		return fmt.Errorf("read foundation migration: %w", err)
	}
	if _, err := r.pool.Exec(ctx, string(script)); err != nil {
		return fmt.Errorf("apply foundation migration: %w", err)
	}
	return nil
}
