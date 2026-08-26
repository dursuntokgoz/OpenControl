package store

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/dursuntokgoz/OpenControl/internal/core"
)

type dbUserRepo struct{ db *sql.DB }

func NewUserRepo(db *sql.DB) core.UserRepo { return &dbUserRepo{db: db} }

const userSelectCols = `id, username, email, password_hash, role,
	COALESCE(totp_secret, ''), totp_enabled,
	COALESCE(contact_email, ''), language, created_at, updated_at, disabled_at`

func (r *dbUserRepo) Create(ctx context.Context, u *core.User) error {
	const q = `INSERT INTO users (username, email, password_hash, role, totp_secret, totp_enabled,
		contact_email, language, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	res, err := r.db.ExecContext(ctx, q,
		u.Username, u.Email, u.PasswordHash, u.Role, nullStr(u.TOTPSecret), u.TOTPEnabled,
		nullStr(u.ContactEmail), u.Language, u.CreatedAt, u.UpdatedAt)
	if err != nil {
		return err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return err
	}
	u.ID = id
	return nil
}

func (r *dbUserRepo) GetByID(ctx context.Context, id int64) (*core.User, error) {
	q := `SELECT ` + userSelectCols + ` FROM users WHERE id = ?`
	u := &core.User{}
	err := r.db.QueryRowContext(ctx, q, id).Scan(
		&u.ID, &u.Username, &u.Email, &u.PasswordHash, &u.Role, &u.TOTPSecret, &u.TOTPEnabled,
		&u.ContactEmail, &u.Language, &u.CreatedAt, &u.UpdatedAt, &u.DisabledAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return u, err
}

func (r *dbUserRepo) GetByUsername(ctx context.Context, username string) (*core.User, error) {
	q := `SELECT ` + userSelectCols + ` FROM users WHERE username = ?`
	u := &core.User{}
	err := r.db.QueryRowContext(ctx, q, username).Scan(
		&u.ID, &u.Username, &u.Email, &u.PasswordHash, &u.Role, &u.TOTPSecret, &u.TOTPEnabled,
		&u.ContactEmail, &u.Language, &u.CreatedAt, &u.UpdatedAt, &u.DisabledAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return u, err
}

func (r *dbUserRepo) GetByEmail(ctx context.Context, email string) (*core.User, error) {
	q := `SELECT ` + userSelectCols + ` FROM users WHERE email = ?`
	u := &core.User{}
	err := r.db.QueryRowContext(ctx, q, email).Scan(
		&u.ID, &u.Username, &u.Email, &u.PasswordHash, &u.Role, &u.TOTPSecret, &u.TOTPEnabled,
		&u.ContactEmail, &u.Language, &u.CreatedAt, &u.UpdatedAt, &u.DisabledAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return u, err
}

func (r *dbUserRepo) Update(ctx context.Context, u *core.User) error {
	const q = `UPDATE users SET email=?, password_hash=?, role=?, totp_secret=?, totp_enabled=?,
		contact_email=?, language=?, updated_at=?, disabled_at=? WHERE id=?`
	u.UpdatedAt = time.Now()
	_, err := r.db.ExecContext(ctx, q,
		u.Email, u.PasswordHash, u.Role, nullStr(u.TOTPSecret), u.TOTPEnabled,
		nullStr(u.ContactEmail), u.Language, u.UpdatedAt, u.DisabledAt, u.ID)
	return err
}

func (r *dbUserRepo) Delete(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM users WHERE id = ?`, id)
	return err
}

func (r *dbUserRepo) List(ctx context.Context, offset, limit int) ([]core.User, int64, error) {
	var total int64
	_ = r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM users`).Scan(&total)
	q := `SELECT ` + userSelectCols + ` FROM users ORDER BY id LIMIT ? OFFSET ?`
	rows, err := r.db.QueryContext(ctx, q, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = rows.Close() }()
	var users []core.User
	for rows.Next() {
		var u core.User
		if err := rows.Scan(
			&u.ID, &u.Username, &u.Email, &u.PasswordHash, &u.Role, &u.TOTPSecret, &u.TOTPEnabled,
			&u.ContactEmail, &u.Language, &u.CreatedAt, &u.UpdatedAt, &u.DisabledAt); err != nil {
			return nil, 0, err
		}
		users = append(users, u)
	}
	return users, total, rows.Err()
}

func (r *dbUserRepo) Count(ctx context.Context) (int64, error) {
	var n int64
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM users`).Scan(&n)
	return n, err
}

// nullStr returns nil for empty strings (stored as NULL in DB).
func nullStr(s string) any {
	if s == "" {
		return nil
	}
	return s
}
