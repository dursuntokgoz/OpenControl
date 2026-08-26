package store

import (
	"context"
	"database/sql"
	"testing"
)

// OpenTestDB creates an in-memory SQLite database with all migrations applied.
// For use in tests only.
func OpenTestDB(t *testing.T) *DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	sdb := &DB{DB: db, Driver: "sqlite"}
	if err := sdb.Migrate(context.Background()); err != nil {
		t.Fatal(err)
	}
	return sdb
}
