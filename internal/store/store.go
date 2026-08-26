// Package store provides the persistence layer: opening SQLite or PostgreSQL
// databases and running embedded goose migrations.
package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	_ "github.com/jackc/pgx/v5/stdlib" // postgres driver
	"github.com/pressly/goose/v3"
	_ "modernc.org/sqlite" // pure-Go sqlite driver

	"github.com/dursuntokgoz/OpenControl/internal/config"
)

// DB wraps sql.DB with ServerPanel helpers.
type DB struct {
	*sql.DB
	Driver string
}

// Open connects to the configured database without running migrations.
func Open(cfg config.Database) (*DB, error) {
	var dsn, driver string
	switch cfg.Driver {
	case "sqlite":
		// _time_format ensures consistent time handling; busy_timeout avoids
		// "database is locked" under concurrent workers.
		dsn = fmt.Sprintf("file:%s?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)", cfg.SQLitePath)
		driver = "sqlite"
	case "postgres":
		dsn = cfg.PostgresDSN
		driver = "pgx"
	default:
		return nil, fmt.Errorf("unsupported driver %q", cfg.Driver)
	}
	db, err := sql.Open(driver, dsn)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", cfg.Driver, err)
	}
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping %s: %w", cfg.Driver, err)
	}
	return &DB{DB: db, Driver: cfg.Driver}, nil
}

// Migrate applies all pending embedded migrations.
func (d *DB) Migrate(ctx context.Context) error {
	goose.SetBaseFS(migrationsFS)
	dialect := "sqlite3"
	if d.Driver == "postgres" {
		dialect = "postgres"
	}
	if err := goose.SetDialect(dialect); err != nil {
		return fmt.Errorf("set dialect: %w", err)
	}
	goose.SetLogger(goose.NopLogger())
	if err := goose.UpContext(ctx, d.DB, "migrations"); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}
	return nil
}

// Close closes the underlying database.
func (d *DB) Close() error { return d.DB.Close() }

// ErrNotFound is returned by lookups that match no rows.
var ErrNotFound = errors.New("not found")
