package store

import (
	"context"
	"database/sql"
	"errors"

	"github.com/dursuntokgoz/OpenControl/internal/core"
)

type dbPackageRepo struct{ db *sql.DB }

func NewPackageRepo(db *sql.DB) core.PackageRepo { return &dbPackageRepo{db: db} }

func (r *dbPackageRepo) Create(ctx context.Context, p *core.Package) error {
	const q = `INSERT INTO packages (name, disk_mb, bandwidth_mb, max_domains, max_databases,
		max_mailboxes, max_ftp, php_version, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`
	res, err := r.db.ExecContext(ctx, q,
		p.Name, p.DiskMB, p.BandwidthMB, p.MaxDomains, p.MaxDatabases,
		p.MaxMailboxes, p.MaxFTP, p.PHPVersion, p.CreatedAt)
	if err != nil {
		return err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return err
	}
	p.ID = id
	return nil
}

func (r *dbPackageRepo) GetByID(ctx context.Context, id int64) (*core.Package, error) {
	const q = `SELECT id, name, disk_mb, bandwidth_mb, max_domains, max_databases,
		max_mailboxes, max_ftp, php_version, created_at FROM packages WHERE id = ?`
	p := &core.Package{}
	err := r.db.QueryRowContext(ctx, q, id).Scan(
		&p.ID, &p.Name, &p.DiskMB, &p.BandwidthMB, &p.MaxDomains, &p.MaxDatabases,
		&p.MaxMailboxes, &p.MaxFTP, &p.PHPVersion, &p.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return p, err
}

func (r *dbPackageRepo) Update(ctx context.Context, p *core.Package) error {
	const q = `UPDATE packages SET name=?, disk_mb=?, bandwidth_mb=?, max_domains=?, max_databases=?,
		max_mailboxes=?, max_ftp=?, php_version=? WHERE id=?`
	_, err := r.db.ExecContext(ctx, q,
		p.Name, p.DiskMB, p.BandwidthMB, p.MaxDomains, p.MaxDatabases,
		p.MaxMailboxes, p.MaxFTP, p.PHPVersion, p.ID)
	return err
}

func (r *dbPackageRepo) Delete(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM packages WHERE id = ?`, id)
	return err
}

func (r *dbPackageRepo) List(ctx context.Context, offset, limit int) ([]core.Package, int64, error) {
	var total int64
	_ = r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM packages`).Scan(&total)
	const q = `SELECT id, name, disk_mb, bandwidth_mb, max_domains, max_databases,
		max_mailboxes, max_ftp, php_version, created_at FROM packages ORDER BY id LIMIT ? OFFSET ?`
	rows, err := r.db.QueryContext(ctx, q, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = rows.Close() }()
	var pkgs []core.Package
	for rows.Next() {
		var p core.Package
		if err := rows.Scan(
			&p.ID, &p.Name, &p.DiskMB, &p.BandwidthMB, &p.MaxDomains, &p.MaxDatabases,
			&p.MaxMailboxes, &p.MaxFTP, &p.PHPVersion, &p.CreatedAt); err != nil {
			return nil, 0, err
		}
		pkgs = append(pkgs, p)
	}
	return pkgs, total, rows.Err()
}
