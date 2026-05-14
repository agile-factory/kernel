package migrator

import (
	"errors"
	"fmt"
	"log/slog"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

// Run applies all pending UP migrations found in migrationsPath.
// migrationsPath must be a file:// URI, e.g. "file://migrations".
func Run(databaseURL, migrationsPath string) error {
	slog.Info("Running database migrations", "source", migrationsPath)

	m, err := migrate.New(migrationsPath, databaseURL)
	if err != nil {
		return fmt.Errorf("migrator: failed to initialize: %w", err)
	}
	defer func() {
		srcErr, dbErr := m.Close()
		if srcErr != nil {
			slog.Error("migrator: failed to close source", "error", srcErr)
		}
		if dbErr != nil {
			slog.Error("migrator: failed to close database", "error", dbErr)
		}
	}()

	if err := m.Up(); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			slog.Info("Migrations: no new migrations to apply")
			return nil
		}
		return fmt.Errorf("migrator: failed to apply migrations: %w", err)
	}

	slog.Info("Migrations applied successfully")
	return nil
}

// Down rolls back all applied migrations (use with caution).
func Down(databaseURL, migrationsPath string) error {
	slog.Warn("Rolling back ALL database migrations", "source", migrationsPath)

	m, err := migrate.New(migrationsPath, databaseURL)
	if err != nil {
		return fmt.Errorf("migrator: failed to initialize: %w", err)
	}
	defer func() {
		srcErr, dbErr := m.Close()
		if srcErr != nil {
			slog.Error("migrator: failed to close source", "error", srcErr)
		}
		if dbErr != nil {
			slog.Error("migrator: failed to close database", "error", dbErr)
		}
	}()

	if err := m.Down(); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			slog.Info("Migrations: nothing to roll back")
			return nil
		}
		return fmt.Errorf("migrator: failed to roll back migrations: %w", err)
	}

	slog.Warn("All migrations rolled back")
	return nil
}
