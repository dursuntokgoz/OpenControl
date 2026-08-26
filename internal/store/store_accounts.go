package store

import (
	"context"
	"database/sql"
	"errors"

	"github.com/dursuntokgoz/OpenControl/internal/core"
)

type dbAccountRepo struct{ db *sql.DB }

func NewAccountRepo(db *sql.DB) core.AccountRepo { return &dbAccountRepo{db: db} }

func (r *dbAccountRepo) Create(ctx context.Context, a *core.Account) error {
	const q = `INSERT INTO accounts (username, primary_domain, owner_user_id, package_id, home_dir,
		status, disk_used_mb, bw_used_mb, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`
	res, err := r.db.ExecContext(ctx, q,
		a.Username, a.PrimaryDomain, a.OwnerUserID, a.PackageID, a.HomeDir,
		a.Status, a.DiskUsedMB, a.BWUsedMB, a.CreatedAt)
	if err != nil {
		return err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return err
	}
	a.ID = id
	return nil
}

func (r *dbAccountRepo) GetByID(ctx context.Context, id int64) (*core.Account, error) {
	const q = `SELECT id, username, primary_domain, owner_user_id, package_id, home_dir,
		status, disk_used_mb, bw_used_mb, created_at FROM accounts WHERE id = ?`
	a := &core.Account{}
	err := r.db.QueryRowContext(ctx, q, id).Scan(
		&a.ID, &a.Username, &a.PrimaryDomain, &a.OwnerUserID, &a.PackageID, &a.HomeDir,
		&a.Status, &a.DiskUsedMB, &a.BWUsedMB, &a.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return a, err
}

func (r *dbAccountRepo) GetByUsername(ctx context.Context, username string) (*core.Account, error) {
	const q = `SELECT id, username, primary_domain, owner_user_id, package_id, home_dir,
		status, disk_used_mb, bw_used_mb, created_at FROM accounts WHERE username = ?`
	a := &core.Account{}
	err := r.db.QueryRowContext(ctx, q, username).Scan(
		&a.ID, &a.Username, &a.PrimaryDomain, &a.OwnerUserID, &a.PackageID, &a.HomeDir,
		&a.Status, &a.DiskUsedMB, &a.BWUsedMB, &a.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return a, err
}

func (r *dbAccountRepo) Update(ctx context.Context, a *core.Account) error {
	const q = `UPDATE accounts SET primary_domain=?, package_id=?, home_dir=?, status=?,
		disk_used_mb=?, bw_used_mb=? WHERE id=?`
	_, err := r.db.ExecContext(ctx, q,
		a.PrimaryDomain, a.PackageID, a.HomeDir, a.Status,
		a.DiskUsedMB, a.BWUsedMB, a.ID)
	return err
}

func (r *dbAccountRepo) Delete(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM accounts WHERE id = ?`, id)
	return err
}

func (r *dbAccountRepo) List(ctx context.Context, offset, limit int) ([]core.Account, int64, error) {
	var total int64
	_ = r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM accounts`).Scan(&total)
	const q = `SELECT id, username, primary_domain, owner_user_id, package_id, home_dir,
		status, disk_used_mb, bw_used_mb, created_at FROM accounts ORDER BY id LIMIT ? OFFSET ?`
	rows, err := r.db.QueryContext(ctx, q, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = rows.Close() }()
	var accounts []core.Account
	for rows.Next() {
		var a core.Account
		if err := rows.Scan(
			&a.ID, &a.Username, &a.PrimaryDomain, &a.OwnerUserID, &a.PackageID, &a.HomeDir,
			&a.Status, &a.DiskUsedMB, &a.BWUsedMB, &a.CreatedAt); err != nil {
			return nil, 0, err
		}
		accounts = append(accounts, a)
	}
	return accounts, total, rows.Err()
}

func (r *dbAccountRepo) ListByOwner(ctx context.Context, ownerID int64, offset, limit int) ([]core.Account, int64, error) {
	var total int64
	_ = r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM accounts WHERE owner_user_id = ?`, ownerID).Scan(&total)
	const q = `SELECT id, username, primary_domain, owner_user_id, package_id, home_dir,
		status, disk_used_mb, bw_used_mb, created_at FROM accounts WHERE owner_user_id = ? ORDER BY id LIMIT ? OFFSET ?`
	rows, err := r.db.QueryContext(ctx, q, ownerID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = rows.Close() }()
	var accounts []core.Account
	for rows.Next() {
		var a core.Account
		if err := rows.Scan(
			&a.ID, &a.Username, &a.PrimaryDomain, &a.OwnerUserID, &a.PackageID, &a.HomeDir,
			&a.Status, &a.DiskUsedMB, &a.BWUsedMB, &a.CreatedAt); err != nil {
			return nil, 0, err
		}
		accounts = append(accounts, a)
	}
	return accounts, total, rows.Err()
}
