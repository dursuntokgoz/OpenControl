// Package providers defines the interfaces every system-service integration
// must implement. Concrete adapters live in subpackages (nginx, apache, ...)
// and are selected at runtime. Interfaces are intentionally small so that
// implementations can be tested against fakes in unit tests.
package providers

import "context"

// ServiceState enumerates systemd unit states we care about.
type ServiceState string

const (
	ServiceActive   ServiceState = "active"
	ServiceInactive ServiceState = "inactive"
	ServiceFailed   ServiceState = "failed"
	ServiceUnknown  ServiceState = "unknown"
)

// ServiceControl manages systemd units via the privileged agent.
type ServiceControl interface {
	// Start starts the unit and waits until it is active.
	Start(ctx context.Context, unit string) error
	// Stop stops the unit.
	Stop(ctx context.Context, unit string) error
	// Restart restarts the unit.
	Restart(ctx context.Context, unit string) error
	// Enable schedules the unit to start on boot.
	Enable(ctx context.Context, unit string) error
	// Disable removes the unit from boot targets.
	Disable(ctx context.Context, unit string) error
	// Status returns coarse-grained state plus the last log lines.
	Status(ctx context.Context, unit string) (ServiceState, []string, error)
}

// WebServerProvider abstracts nginx/Apache configuration management.
type WebServerProvider interface {
	// Name returns "nginx" or "apache".
	Name() string
	// RenderVhost writes a validated virtual host config for the domain.
	RenderVhost(ctx context.Context, domain Vhost) error
	// RemoveVhost removes the vhost config atomically.
	RemoveVhost(ctx context.Context, domain string) error
	// ListVhosts returns configured primary server names.
	ListVhosts(ctx context.Context) ([]string, error)
	// Reload validates configuration then reloads the service.
	Reload(ctx context.Context) error
}

// Vhost is the declarative description of a virtual host.
type Vhost struct {
	Domain       string   `json:"domain"`
	Aliases      []string `json:"aliases,omitempty"`
	DocumentRoot string   `json:"documentRoot"`
	PHPSocket    string   `json:"phpSocket,omitempty"`
	ForceHTTPS   bool     `json:"forceHttps,omitempty"`
	TLSCertPath  string   `json:"tlsCertPath,omitempty"`
	TLSKeyPath   string   `json:"tlsKeyPath,omitempty"`
}

// PHPProvider manages PHP-FPM pools and per-domain versions.
type PHPProvider interface {
	// Versions lists installed PHP versions (e.g. "8.2").
	Versions(ctx context.Context) ([]string, error)
	// EnsurePool creates or updates the FPM pool for an account/domain.
	EnsurePool(ctx context.Context, pool PoolSpec) error
	// RemovePool deletes the pool when the last usage goes away.
	RemovePool(ctx context.Context, poolName string) error
	// Reload applies pool changes after validation.
	Reload(ctx context.Context, version string) error
}

// PoolSpec declares a PHP-FPM pool.
type PoolSpec struct {
	Name       string   `json:"name"`
	Version    string   `json:"version"`
	SocketDir  string   `json:"socketDir"`
	User       string   `json:"user"`
	Group      string   `json:"group"`
	DocRoots   []string `json:"docRoots"`
	ProcessMax int      `json:"processMax"`
}

// DNSZoneProvider manages DNS zones through BIND/PowerDNS.
type DNSZoneProvider interface {
	// WriteZone renders and validates a zone file, then loads it.
	WriteZone(ctx context.Context, zone Zone) error
	// DeleteZone removes the zone.
	DeleteZone(ctx context.Context, origin string) error
	// ListZones returns configured zone origins.
	ListZones(ctx context.Context) ([]string, error)
}

// RecordType enumerates supported resource record types.
type RecordType string

const (
	RecordA     RecordType = "A"
	RecordAAAA  RecordType = "AAAA"
	RecordCNAME RecordType = "CNAME"
	RecordMX    RecordType = "MX"
	RecordTXT   RecordType = "TXT"
	RecordSRV   RecordType = "SRV"
	RecordCAA   RecordType = "CAA"
	RecordNS    RecordType = "NS"
)

// Zone is a DNS zone with its records.
type Zone struct {
	Origin  string   `json:"origin"`
	TTL     uint32   `json:"ttl"`
	NSNames []string `json:"nsNames"`
	Records []Record `json:"records"`
}

// Record is a single resource record.
type Record struct {
	Name     string     `json:"name"`
	Type     RecordType `json:"type"`
	TTL      uint32     `json:"ttl"`
	Priority uint16     `json:"priority,omitempty"`
	Value    string     `json:"value"`
}

// MailProvider manages Postfix/Dovecot mail accounts and routing.
type MailProvider interface {
	CreateAccount(ctx context.Context, acct MailAccount) error
	DeleteAccount(ctx context.Context, address string) error
	CreateAlias(ctx context.Context, source, destination string) error
	DeleteAlias(ctx context.Context, source string) error
	CreateForwarder(ctx context.Context, source, destination string) error
	DeleteForwarder(ctx context.Context, source string) error
	SetQuota(ctx context.Context, address string, quotaMB int64) error
}

// MailAccount is a full mailbox.
type MailAccount struct {
	Address      string `json:"address"`
	PasswordHash string `json:"passwordHash"`
	QuotaMB      int64  `json:"quotaMb"`
	SendOnly     bool   `json:"sendOnly,omitempty"`
}

// DatabaseProvider manages MySQL/MariaDB/PostgreSQL instances.
type DatabaseProvider interface {
	CreateDatabase(ctx context.Context, name string, charset string) error
	DeleteDatabase(ctx context.Context, name string) error
	CreateUser(ctx context.Context, user, password, host string) error
	DeleteUser(ctx context.Context, user, host string) error
	Grant(ctx context.Context, user, host, database string, privileges []string) error
	RemoteAccess(ctx context.Context, database string, allowedHosts []string) error
}

// FirewallProvider abstracts nftables/firewalld rule management.
type FirewallProvider interface {
	AllowPort(ctx context.Context, port int, proto string, sourceCIDR string) error
	DenyPort(ctx context.Context, port int, proto string, sourceCIDR string) error
	ListRules(ctx context.Context) ([]FirewallRule, error)
	BanIP(ctx context.Context, ip string, durationSec int) error
}

// FirewallRule is one persisted firewall entry.
type FirewallRule struct {
	Port    int    `json:"port"`
	Proto   string `json:"proto"`
	Source  string `json:"source,omitempty"`
	Action  string `json:"action"` // allow|deny
	Comment string `json:"comment,omitempty"`
}

// SSLProvider issues and renews certificates.
type SSLProvider interface {
	// Issue obtains a certificate for the identifiers using the given challenge.
	Issue(ctx context.Context, req CertRequest) (*CertBundle, error)
	// Renew forces renewal even if not near expiry.
	Renew(ctx context.Context, domains []string) (*CertBundle, error)
	// Install places cert/key files and wires them into the web server.
	Install(ctx context.Context, domain string, bundle CertBundle) error
}

// ChallengeType selects ACME challenge kind.
type ChallengeType string

const (
	ChallengeHTTP01 ChallengeType = "http-01"
	ChallengeDNS01  ChallengeType = "dns-01"
)

// CertRequest describes an order.
type CertRequest struct {
	Domains   []string      `json:"domains"`
	Challenge ChallengeType `json:"challenge"`
	Email     string        `json:"email"`
}

// CertBundle is an issued certificate with chain and key paths.
type CertBundle struct {
	CertPath  string `json:"certPath"`
	KeyPath   string `json:"keyPath"`
	ChainPath string `json:"chainPath"`
	NotAfter  string `json:"notAfter"`
}
