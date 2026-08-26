-- Migration 00002: authentication, authorization, hosting accounts, audit.
-- +goose Up
CREATE TABLE IF NOT EXISTS users (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    username      TEXT NOT NULL UNIQUE,
    email         TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    role          TEXT NOT NULL CHECK (role IN ('admin','reseller','user')),
    totp_secret   TEXT,
    totp_enabled  INTEGER NOT NULL DEFAULT 0,
    contact_email TEXT,
    language      TEXT NOT NULL DEFAULT 'en',
    created_at    TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at    TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    disabled_at   TIMESTAMP
);

CREATE TABLE IF NOT EXISTS sessions (
    id           TEXT PRIMARY KEY,
    user_id      INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    csrf_token   TEXT NOT NULL,
    ip           TEXT NOT NULL DEFAULT '',
    user_agent   TEXT NOT NULL DEFAULT '',
    created_at   TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at   TIMESTAMP NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_sessions_user ON sessions(user_id);

CREATE TABLE IF NOT EXISTS audit_log (
    id             INTEGER PRIMARY KEY AUTOINCREMENT,
    at             TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    actor_user_id  INTEGER,
    actor_username TEXT NOT NULL DEFAULT '',
    actor_ip       TEXT NOT NULL DEFAULT '',
    action         TEXT NOT NULL,
    target_type    TEXT NOT NULL DEFAULT '',
    target_id      TEXT NOT NULL DEFAULT '',
    detail         TEXT NOT NULL DEFAULT '{}'
);
CREATE INDEX IF NOT EXISTS idx_audit_at ON audit_log(at);

CREATE TABLE IF NOT EXISTS packages (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    name          TEXT NOT NULL UNIQUE,
    disk_mb       INTEGER NOT NULL,
    bandwidth_mb  INTEGER NOT NULL,
    max_domains   INTEGER NOT NULL,
    max_databases INTEGER NOT NULL,
    max_mailboxes INTEGER NOT NULL,
    max_ftp       INTEGER NOT NULL,
    php_version   TEXT NOT NULL DEFAULT 'default',
    created_at    TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS accounts (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    username      TEXT NOT NULL UNIQUE,
    primary_domain TEXT NOT NULL UNIQUE,
    owner_user_id INTEGER NOT NULL REFERENCES users(id),
    package_id    INTEGER NOT NULL REFERENCES packages(id),
    home_dir      TEXT NOT NULL,
    status        TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active','suspended')),
    disk_used_mb  INTEGER NOT NULL DEFAULT 0,
    bw_used_mb    INTEGER NOT NULL DEFAULT 0,
    created_at    TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_accounts_owner ON accounts(owner_user_id);

CREATE TABLE IF NOT EXISTS api_tokens (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id      INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name         TEXT NOT NULL,
    token_hash   TEXT NOT NULL UNIQUE,
    scopes       TEXT NOT NULL DEFAULT '[]',
    created_at   TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_used_at TIMESTAMP,
    expires_at   TIMESTAMP
);

-- +goose Down
DROP TABLE IF EXISTS api_tokens;
DROP TABLE IF EXISTS accounts;
DROP TABLE IF EXISTS packages;
DROP TABLE IF EXISTS audit_log;
DROP TABLE IF EXISTS sessions;
DROP TABLE IF EXISTS users;
