package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultIsValid(t *testing.T) {
	cfg := Default()
	if err := cfg.Validate(); err != nil {
		t.Fatalf("default config invalid: %v", err)
	}
	if cfg.HTTP.Listen != DefaultHTTPListen {
		t.Errorf("listen = %q, want %q", cfg.HTTP.Listen, DefaultHTTPListen)
	}
	if cfg.Database.Driver != "sqlite" {
		t.Errorf("driver = %q, want sqlite", cfg.Database.Driver)
	}
}

func TestLoadFromYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	content := `
http:
  listen: 0.0.0.0:9000
database:
  driver: postgres
  postgresDsn: postgres://u:p@localhost/db
agent:
  socketPath: /run/sp/agent.sock
  token: sekrit
log:
  format: json
`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.HTTP.Listen != "0.0.0.0:9000" {
		t.Errorf("listen = %q", cfg.HTTP.Listen)
	}
	if cfg.Database.PostgresDSN != "postgres://u:p@localhost/db" {
		t.Errorf("dsn = %q", cfg.Database.PostgresDSN)
	}
	if cfg.Agent.Token != "sekrit" {
		t.Errorf("token = %q", cfg.Agent.Token)
	}
}

func TestLoadMissingFile(t *testing.T) {
	if _, err := Load(filepath.Join(t.TempDir(), "nope.yaml")); err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestEnvOverrides(t *testing.T) {
	t.Setenv("SERVERPANEL_HTTP_LISTEN", "127.0.0.1:9999")
	t.Setenv("SERVERPANEL_AGENT_TOKEN", "envtok")
	cfg := Load_or_default(t)
	if cfg.HTTP.Listen != "127.0.0.1:9999" {
		t.Errorf("listen = %q", cfg.HTTP.Listen)
	}
	if cfg.Agent.Token != "envtok" {
		t.Errorf("token = %q", cfg.Agent.Token)
	}
}

func TestValidateRejectsBadDriver(t *testing.T) {
	cfg := Default()
	cfg.Database.Driver = "oracle"
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error for unsupported driver")
	}
}

func TestValidatePostgresRequiresDSN(t *testing.T) {
	cfg := Default()
	cfg.Database.Driver = "postgres"
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error for missing postgres DSN")
	}
	cfg.Database.PostgresDSN = "postgres://x"
	if err := cfg.Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateRelativeAgentSocketRejected(t *testing.T) {
	cfg := Default()
	cfg.Agent.SocketPath = "relative.sock"
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error for relative socket path")
	}
}

func TestValidateBadLogFormat(t *testing.T) {
	cfg := Default()
	cfg.Log.Format = "xml"
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error for bad log format")
	}
}

// helper keeps tests readable
func Load_or_default(t *testing.T) *Config {
	t.Helper()
	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Load empty: %v", err)
	}
	return cfg
}
