package store

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/dursuntokgoz/OpenControl/internal/config"
)

func testSQLiteDB(t *testing.T) *DB {
	t.Helper()
	dir := t.TempDir()
	db, err := Open(config.Database{
		Driver:     "sqlite",
		SQLitePath: filepath.Join(dir, "test.db"),
	})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func TestMigrateIsIdempotent(t *testing.T) {
	db := testSQLiteDB(t)
	ctx := context.Background()
	for i := 0; i < 2; i++ {
		if err := db.Migrate(ctx); err != nil {
			t.Fatalf("migrate pass %d: %v", i+1, err)
		}
	}
	var version int64
	if err := db.QueryRowContext(ctx,
		`SELECT max(version_id) FROM goose_db_version`).Scan(&version); err != nil {
		t.Fatalf("read version: %v", err)
	}
	if version < 1 {
		t.Fatalf("version = %d, want >= 1", version)
	}
}

func TestSettingsRoundtrip(t *testing.T) {
	db := testSQLiteDB(t)
	ctx := context.Background()
	if err := db.Migrate(ctx); err != nil {
		t.Fatal(err)
	}

	if _, err := db.GetSetting(ctx, "missing"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
	if err := db.SetSetting(ctx, "install.completed", "true"); err != nil {
		t.Fatalf("set: %v", err)
	}
	got, err := db.GetSetting(ctx, "install.completed")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got != "true" {
		t.Fatalf("value = %q, want true", got)
	}
	if err := db.SetSetting(ctx, "install.completed", "false"); err != nil {
		t.Fatalf("overwrite: %v", err)
	}
	got, _ = db.GetSetting(ctx, "install.completed")
	if got != "false" {
		t.Fatalf("after overwrite value = %q", got)
	}
}

func TestOpenUnsupportedDriver(t *testing.T) {
	if _, err := Open(config.Database{Driver: "oracle"}); err == nil {
		t.Fatal("expected error")
	}
}
