package store

import (
	"context"
	"database/sql"

	"github.com/dursuntokgoz/OpenControl/internal/core"
)

type dbAuditRepo struct{ db *sql.DB }

func NewAuditRepo(db *sql.DB) core.AuditRepo { return &dbAuditRepo{db: db} }

func (r *dbAuditRepo) Write(ctx context.Context, entry *core.AuditEntry) error {
	const q = `INSERT INTO audit_log (at, actor_user_id, actor_username, actor_ip, action, target_type, target_id, detail)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`
	_, err := r.db.ExecContext(ctx, q,
		entry.At, entry.ActorUserID, entry.ActorUsername, entry.ActorIP,
		entry.Action, entry.TargetType, entry.TargetID, entry.Detail)
	return err
}

func (r *dbAuditRepo) List(ctx context.Context, offset, limit int) ([]core.AuditEntry, int64, error) {
	var total int64
	_ = r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM audit_log`).Scan(&total)
	const q = `SELECT id, at, actor_user_id, actor_username, actor_ip, action, target_type, target_id, detail
		FROM audit_log ORDER BY id DESC LIMIT ? OFFSET ?`
	rows, err := r.db.QueryContext(ctx, q, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = rows.Close() }()
	var entries []core.AuditEntry
	for rows.Next() {
		var e core.AuditEntry
		if err := rows.Scan(&e.ID, &e.At, &e.ActorUserID, &e.ActorUsername, &e.ActorIP,
			&e.Action, &e.TargetType, &e.TargetID, &e.Detail); err != nil {
			return nil, 0, err
		}
		entries = append(entries, e)
	}
	return entries, total, rows.Err()
}
