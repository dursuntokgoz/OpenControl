package store

import (
	"context"
	"database/sql"
	"time"

	"github.com/dursuntokgoz/OpenControl/internal/auth"
)

type dbSessionStore struct{ db *sql.DB }

func NewSessionStore(db *sql.DB) auth.SessionStore { return &dbSessionStore{db: db} }

func (s *dbSessionStore) CreateSession(ctx context.Context, sess *auth.Session) error {
	const q = `INSERT INTO sessions (id, user_id, csrf_token, ip, user_agent, created_at, expires_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`
	_, err := s.db.ExecContext(ctx, q, sess.ID, sess.UserID, sess.CSRFToken, sess.IP, sess.UserAgent, sess.CreatedAt, sess.ExpiresAt)
	return err
}

func (s *dbSessionStore) GetSession(ctx context.Context, id string) (*auth.Session, error) {
	const q = `SELECT id, user_id, csrf_token, created_at, expires_at, ip, user_agent FROM sessions WHERE id = ?`
	sess := &auth.Session{}
	err := s.db.QueryRowContext(ctx, q, id).Scan(&sess.ID, &sess.UserID, &sess.CSRFToken, &sess.CreatedAt, &sess.ExpiresAt, &sess.IP, &sess.UserAgent)
	return sess, err
}

func (s *dbSessionStore) DeleteSession(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM sessions WHERE id = ?`, id)
	return err
}

func (s *dbSessionStore) DeleteExpiredSessions(ctx context.Context) (int64, error) {
	r, err := s.db.ExecContext(ctx, `DELETE FROM sessions WHERE expires_at < ?`, time.Now())
	if err != nil {
		return 0, err
	}
	return r.RowsAffected()
}
