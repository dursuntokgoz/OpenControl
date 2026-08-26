// Package config loads ServerPanel configuration from YAML files and
// environment variables. Secrets are never hardcoded; they come from the
// config file (/etc/serverpanel/config.yaml) or the environment.
package config

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// Defaults used when a value is absent from the config file and environment.
const (
	DefaultHTTPListen    = "127.0.0.1:8080"
	DefaultSQLitePath    = "/var/lib/serverpanel/panel.db"
	DefaultAgentSocket   = "/run/serverpanel/agent.sock"
	DefaultWebDist       = "/usr/share/serverpanel/web"
	DefaultSessionSecret = ""
	DefaultPostgresDSN   = ""
	DefaultLogFormatText = "text"
)

// Database selects and configures the backing store.
type Database struct {
	// Driver is either "sqlite" (default) or "postgres".
	Driver string `yaml:"driver"`
	// SQLitePath is the file path used when Driver == "sqlite".
	SQLitePath string `yaml:"sqlitePath"`
	// PostgresDSN is the DSN used when Driver == "postgres".
	PostgresDSN string `yaml:"postgresDsn"`
}

// HTTP holds the public API server settings.
type HTTP struct {
	Listen string `yaml:"listen"`
	// WebDist is the directory containing the built React application.
	WebDist string `yaml:"webDist"`
}

// Agent describes how to reach the privileged panel-agent.
type Agent struct {
	SocketPath string `yaml:"socketPath"`
	// Token is the shared secret presented to the agent on every call.
	Token string `yaml:"token"`
}

// Log controls logging output.
type Log struct {
	// Format is "text" or "json".
	Format string `yaml:"format"`
}

// Config is the root configuration object.
type Config struct {
	HTTP     HTTP     `yaml:"http"`
	Database Database `yaml:"database"`
	Agent    Agent    `yaml:"agent"`
	Log      Log      `yaml:"log"`

	// SessionSecret signs session cookies; generate per-install.
	SessionSecret string `yaml:"sessionSecret"`
}

// Load reads configuration from path (when non-empty), applies environment
// overrides (SERVERPANEL_* variables) and returns the resulting Config.
func Load(path string) (*Config, error) {
	cfg := &Config{}
	if path != "" {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read config %s: %w", path, err)
		}
		if err := yaml.Unmarshal(data, cfg); err != nil {
			return nil, fmt.Errorf("parse config %s: %w", path, err)
		}
	}
	applyDefaults(cfg)
	applyEnv(cfg)
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

// Default returns the built-in development configuration.
func Default() *Config {
	cfg := &Config{}
	applyDefaults(cfg)
	return cfg
}

func applyDefaults(cfg *Config) {
	if cfg.HTTP.Listen == "" {
		cfg.HTTP.Listen = DefaultHTTPListen
	}
	if cfg.HTTP.WebDist == "" {
		cfg.HTTP.WebDist = DefaultWebDist
	}
	if cfg.Database.Driver == "" {
		cfg.Database.Driver = "sqlite"
	}
	if cfg.Database.SQLitePath == "" {
		cfg.Database.SQLitePath = DefaultSQLitePath
	}
	if cfg.Agent.SocketPath == "" {
		cfg.Agent.SocketPath = DefaultAgentSocket
	}
	if cfg.Log.Format == "" {
		cfg.Log.Format = DefaultLogFormatText
	}
}

func applyEnv(cfg *Config) {
	setStr := func(dst *string, env string) {
		if v, ok := os.LookupEnv(env); ok && v != "" {
			*dst = v
		}
	}
	setStr(&cfg.HTTP.Listen, "SERVERPANEL_HTTP_LISTEN")
	setStr(&cfg.HTTP.WebDist, "SERVERPANEL_WEB_DIST")
	setStr(&cfg.Database.Driver, "SERVERPANEL_DB_DRIVER")
	setStr(&cfg.Database.SQLitePath, "SERVERPANEL_DB_SQLITE_PATH")
	setStr(&cfg.Database.PostgresDSN, "SERVERPANEL_DB_POSTGRES_DSN")
	setStr(&cfg.Agent.SocketPath, "SERVERPANEL_AGENT_SOCKET")
	setStr(&cfg.Agent.Token, "SERVERPANEL_AGENT_TOKEN")
	setStr(&cfg.SessionSecret, "SERVERPANEL_SESSION_SECRET")
	setStr(&cfg.Log.Format, "SERVERPANEL_LOG_FORMAT")
}

// Validate enforces invariants that must hold before the server starts.
func (c *Config) Validate() error {
	switch strings.ToLower(c.Database.Driver) {
	case "sqlite":
		if c.Database.SQLitePath == "" {
			return fmt.Errorf("database.sqlitePath is required for sqlite driver")
		}
	case "postgres":
		if c.Database.PostgresDSN == "" {
			return fmt.Errorf("database.postgresDsn is required for postgres driver")
		}
	default:
		return fmt.Errorf("unsupported database driver %q (want sqlite or postgres)", c.Database.Driver)
	}
	if c.Agent.SocketPath == "" {
		return fmt.Errorf("agent.socketPath is required")
	}
	// Paths refer to the Linux server filesystem regardless of the build host,
	// so validate POSIX-absoluteness explicitly instead of filepath.IsAbs.
	if !strings.HasPrefix(c.Agent.SocketPath, "/") {
		return fmt.Errorf("agent.socketPath must be an absolute POSIX path, got %q", c.Agent.SocketPath)
	}
	switch strings.ToLower(c.Log.Format) {
	case "text", "json":
	default:
		return fmt.Errorf("unsupported log format %q (want text or json)", c.Log.Format)
	}
	return nil
}
