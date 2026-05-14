// Package migrator runs SQL migrations embedded in platform packages.
//
// Each platform exports its own //go:embed-ed migrations FS. The consumer
// composes them via Run, in dependency order. Each Source gets its own
// version-tracking table (schema_migrations_<name>) so packages can evolve
// their migration history independently.
package migrator

import (
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"net/url"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

// Source binds an embedded migrations directory to a logical name. The Name
// is used to scope the version-tracking table (schema_migrations_<name>).
// Path is the directory inside FS where SQL files live, e.g. "migrations".
type Source struct {
	Name string
	FS   fs.FS
	Path string
}

// Run applies all pending UP migrations for each source, in order. Each
// source is tracked in its own schema_migrations_<name> table so packages
// can evolve their migrations independently.
func Run(databaseURL string, sources []Source) error {
	for _, src := range sources {
		if err := runOne(databaseURL, src, upDirection); err != nil {
			return err
		}
	}
	return nil
}

// Down rolls back all applied migrations for each source, in REVERSE order.
// Use with caution — intended for dev resets.
func Down(databaseURL string, sources []Source) error {
	for i := len(sources) - 1; i >= 0; i-- {
		if err := runOne(databaseURL, sources[i], downDirection); err != nil {
			return err
		}
	}
	return nil
}

type direction int

const (
	upDirection direction = iota
	downDirection
)

func runOne(databaseURL string, src Source, dir direction) error {
	if src.Name == "" {
		return fmt.Errorf("migrator: source has empty name")
	}
	path := src.Path
	if path == "" {
		path = "migrations"
	}

	srcDriver, err := iofs.New(src.FS, path)
	if err != nil {
		return fmt.Errorf("migrator: %s: iofs new: %w", src.Name, err)
	}

	dbURL, err := withMigrationsTable(databaseURL, "schema_migrations_"+src.Name)
	if err != nil {
		return fmt.Errorf("migrator: %s: build url: %w", src.Name, err)
	}

	m, err := migrate.NewWithSourceInstance(src.Name, srcDriver, dbURL)
	if err != nil {
		return fmt.Errorf("migrator: %s: init: %w", src.Name, err)
	}
	defer func() {
		srcErr, dbErr := m.Close()
		if srcErr != nil {
			slog.Error("migrator: failed to close source", "source", src.Name, "error", srcErr)
		}
		if dbErr != nil {
			slog.Error("migrator: failed to close database", "source", src.Name, "error", dbErr)
		}
	}()

	var runErr error
	switch dir {
	case upDirection:
		slog.Info("migrator: applying up", "source", src.Name)
		runErr = m.Up()
	case downDirection:
		slog.Warn("migrator: rolling back", "source", src.Name)
		runErr = m.Down()
	}

	if runErr != nil {
		if errors.Is(runErr, migrate.ErrNoChange) {
			slog.Info("migrator: no change", "source", src.Name)
			return nil
		}
		return fmt.Errorf("migrator: %s: %w", src.Name, runErr)
	}
	slog.Info("migrator: applied", "source", src.Name)
	return nil
}

// withMigrationsTable adds the x-migrations-table query parameter to the
// postgres driver URL so this source's history lives in its own table.
func withMigrationsTable(databaseURL, table string) (string, error) {
	u, err := url.Parse(databaseURL)
	if err != nil {
		return "", err
	}
	q := u.Query()
	q.Set("x-migrations-table", table)
	u.RawQuery = q.Encode()
	return u.String(), nil
}
