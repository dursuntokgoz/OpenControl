package core

import (
	"context"
	"time"
)

// User is an authenticated panel user.
type User struct {
	ID           int64      `json:"id"`
	Username     string     `json:"username"`
	Email        string     `json:"email"`
	PasswordHash string     `json:"-"`
	Role         string     `json:"role"` // admin, reseller, user
	TOTPSecret   string     `json:"-"`
	TOTPEnabled  bool       `json:"totpEnabled"`
	ContactEmail string     `json:"contactEmail,omitempty"`
	Language     string     `json:"language"`
	CreatedAt    time.Time  `json:"createdAt"`
	UpdatedAt    time.Time  `json:"updatedAt"`
	DisabledAt   *time.Time `json:"disabledAt,omitempty"`
}

func (u *User) IsAdmin() bool    { return u.Role == "admin" }
func (u *User) IsReseller() bool { return u.Role == "reseller" }

// UserRepo defines the persistence interface for users.
type UserRepo interface {
	Create(ctx context.Context, u *User) error
	GetByID(ctx context.Context, id int64) (*User, error)
	GetByUsername(ctx context.Context, username string) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	Update(ctx context.Context, u *User) error
	Delete(ctx context.Context, id int64) error
	List(ctx context.Context, offset, limit int) ([]User, int64, error)
	Count(ctx context.Context) (int64, error)
}

// AuditEntry is an immutable audit log line.
type AuditEntry struct {
	ID            int64     `json:"id"`
	At            time.Time `json:"at"`
	ActorUserID   *int64    `json:"actorUserId,omitempty"`
	ActorUsername string    `json:"actorUsername"`
	ActorIP       string    `json:"actorIp"`
	Action        string    `json:"action"`
	TargetType    string    `json:"targetType"`
	TargetID      string    `json:"targetId"`
	Detail        string    `json:"detail"`
}

// AuditRepo defines the persistence interface for audit log entries.
type AuditRepo interface {
	Write(ctx context.Context, entry *AuditEntry) error
	List(ctx context.Context, offset, limit int) ([]AuditEntry, int64, error)
}

// Package is a hosting plan with resource limits.
type Package struct {
	ID           int64     `json:"id"`
	Name         string    `json:"name"`
	DiskMB       int64     `json:"diskMb"`
	BandwidthMB  int64     `json:"bandwidthMb"`
	MaxDomains   int       `json:"maxDomains"`
	MaxDatabases int       `json:"maxDatabases"`
	MaxMailboxes int       `json:"maxMailboxes"`
	MaxFTP       int       `json:"maxFtp"`
	PHPVersion   string    `json:"phpVersion"`
	CreatedAt    time.Time `json:"createdAt"`
}

// PackageRepo defines the persistence interface for hosting packages.
type PackageRepo interface {
	Create(ctx context.Context, p *Package) error
	GetByID(ctx context.Context, id int64) (*Package, error)
	Update(ctx context.Context, p *Package) error
	Delete(ctx context.Context, id int64) error
	List(ctx context.Context, offset, limit int) ([]Package, int64, error)
}

// Account is a hosting account (cPanel-like user).
type Account struct {
	ID            int64     `json:"id"`
	Username      string    `json:"username"`
	PrimaryDomain string    `json:"primaryDomain"`
	OwnerUserID   int64     `json:"ownerUserId"`
	PackageID     int64     `json:"packageId"`
	HomeDir       string    `json:"homeDir"`
	Status        string    `json:"status"` // active, suspended
	DiskUsedMB    int64     `json:"diskUsedMb"`
	BWUsedMB      int64     `json:"bwUsedMb"`
	CreatedAt     time.Time `json:"createdAt"`
}

// AccountRepo defines the persistence interface for hosting accounts.
type AccountRepo interface {
	Create(ctx context.Context, a *Account) error
	GetByID(ctx context.Context, id int64) (*Account, error)
	GetByUsername(ctx context.Context, username string) (*Account, error)
	Update(ctx context.Context, a *Account) error
	Delete(ctx context.Context, id int64) error
	List(ctx context.Context, offset, limit int) ([]Account, int64, error)
	ListByOwner(ctx context.Context, ownerID int64, offset, limit int) ([]Account, int64, error)
}

// APIToken is a scoped bearer token for API access.
type APIToken struct {
	ID         int64      `json:"id"`
	UserID     int64      `json:"userId"`
	Name       string     `json:"name"`
	TokenHash  string     `json:"-"`
	Scopes     string     `json:"scopes"`
	CreatedAt  time.Time  `json:"createdAt"`
	LastUsedAt *time.Time `json:"lastUsedAt,omitempty"`
	ExpiresAt  *time.Time `json:"expiresAt,omitempty"`
}

// AppDeps holds all dependencies injected into the HTTP layer.
type AppDeps struct {
	Users    UserRepo
	Audit    AuditRepo
	Packages PackageRepo
	Accounts AccountRepo
}
