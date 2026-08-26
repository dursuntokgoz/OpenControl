-- Migration 00001: base metadata tables.
-- +goose Up
CREATE TABLE IF NOT EXISTS panel_settings (
    key         TEXT PRIMARY KEY,
    value       TEXT NOT NULL,
    updated_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- +goose Down
DROP TABLE IF EXISTS panel_settings;
