package store

import (
	"context"
	"database/sql"
	"errors"
)

// Settings are simple key/value runtime settings persisted in the database.
type Setting struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// SetSetting upserts a setting.
func (d *DB) SetSetting(ctx context.Context, key, value string) error {
	const q = `INSERT INTO panel_settings (key, value) VALUES (?, ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value`
	_, err := d.ExecContext(ctx, q, key, value)
	return err
}

// GetSetting reads a setting; returns ErrNotFound when absent.
func (d *DB) GetSetting(ctx context.Context, key string) (string, error) {
	const q = `SELECT value FROM panel_settings WHERE key = ?`
	var value string
	err := d.QueryRowContext(ctx, q, key).Scan(&value)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrNotFound
	}
	return value, err
}
