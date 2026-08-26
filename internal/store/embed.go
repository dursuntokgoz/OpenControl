package store

import "embed"

// migrationsFS embeds SQL migration files so binaries are self-contained.
//
//go:embed migrations/*.sql
var migrationsFS embed.FS
