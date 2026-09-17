// Command panelctl is the operator CLI: install bootstrap, account
// management and diagnostics.
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"syscall"
	"time"

	"github.com/dursuntokgoz/OpenControl/internal/agent"
	"github.com/dursuntokgoz/OpenControl/internal/auth"
	"github.com/dursuntokgoz/OpenControl/internal/config"
	"github.com/dursuntokgoz/OpenControl/internal/core"
	"github.com/dursuntokgoz/OpenControl/internal/store"
)

const usage = `panelctl — ServerPanel control CLI

Usage:
  panelctl version                      Show build information
  panelctl doctor [-config PATH]        Diagnose configuration, database, agent and API
  panelctl agent-ping                   Verify the privileged agent responds
  panelctl bootstrap-admin              Create the initial admin user
`

func main() {
	if len(os.Args) < 2 {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(2)
	}
	var err error
	switch os.Args[1] {
	case "version":
		err = cmdVersion(os.Stdout)
	case "doctor":
		fs := flag.NewFlagSet("doctor", flag.ExitOnError)
		configPath := fs.String("config", os.Getenv("SERVERPANEL_CONFIG"), "path to config.yaml")
		_ = fs.Parse(os.Args[2:])
		err = cmdDoctor(context.Background(), *configPath)
	case "agent-ping":
		fs := flag.NewFlagSet("agent-ping", flag.ExitOnError)
		configPath := fs.String("config", os.Getenv("SERVERPANEL_CONFIG"), "path to config.yaml")
		_ = fs.Parse(os.Args[2:])
		err = cmdAgentPing(context.Background(), *configPath)
	case "bootstrap-admin":
		fs := flag.NewFlagSet("bootstrap-admin", flag.ExitOnError)
		configPath := fs.String("config", os.Getenv("SERVERPANEL_CONFIG"), "path to config.yaml")
		username := fs.String("username", "", "admin username (interactive if empty)")
		password := fs.String("password", "", "admin password (interactive if empty)")
		_ = fs.Parse(os.Args[2:])
		err = cmdBootstrapAdmin(context.Background(), *configPath, *username, *password)
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n%s", os.Args[1], usage)
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func cmdVersion(w io.Writer) error {
	info := core.CurrentBuild()
	_, err := fmt.Fprintln(w, info.String())
	return err
}

type doctorCheck struct {
	Name     string `json:"name"`
	OK       bool   `json:"ok"`
	Detail   string `json:"detail"`
	Critical bool   `json:"critical"`
}

func cmdDoctor(ctx context.Context, configPath string) error {
	cfg, err := config.Load(configPath)
	checks := []doctorCheck{{
		Name: "config", OK: err == nil,
		Detail: statusDetail(err), Critical: true,
	}}
	if err != nil {
		return reportDoctor(checks)
	}

	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	st, storeErr := store.Open(cfg.Database)
	storeOK := storeErr == nil
	detail := statusDetail(storeErr)
	if storeOK {
		if mErr := st.Migrate(ctx); mErr != nil {
			storeOK, detail = false, "migrate: "+mErr.Error()
		} else {
			detail = "database reachable, migrations applied"
			if closeErr := st.Close(); closeErr != nil {
				detail += " (close warning: " + closeErr.Error() + ")"
			}
		}
	}
	checks = append(checks, doctorCheck{Name: "database", OK: storeOK, Detail: detail, Critical: true})

	client := agent.NewClient(cfg.Agent.SocketPath, cfg.Agent.Token)
	pingCtx, pingCancel := context.WithTimeout(ctx, 5*time.Second)
	defer pingCancel()
	res, pingErr := client.Ping(pingCtx)
	checks = append(checks, doctorCheck{
		Name: "agent",
		OK:   pingErr == nil && res != nil && res.Pong,
		Detail: func() string {
			if pingErr == nil && res != nil {
				return "agent alive at " + res.Host
			}
			return statusDetail(pingErr)
		}(),
		Critical: true,
	})

	apiURL := fmt.Sprintf("http://%s/healthz", cfg.HTTP.Listen)
	httpClient := &http.Client{Timeout: 5 * time.Second}
	resp, apiErr := httpClient.Get(apiURL)
	apiOK := apiErr == nil && resp.StatusCode == http.StatusOK
	apiDetail := statusDetail(apiErr)
	if apiOK {
		apiDetail = "api healthy on " + cfg.HTTP.Listen
	}
	if resp != nil {
		_ = resp.Body.Close()
	}
	checks = append(checks, doctorCheck{Name: "api", OK: apiOK, Detail: apiDetail, Critical: false})

	return reportDoctor(checks)
}

func cmdAgentPing(ctx context.Context, configPath string) error {
	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}
	client := agent.NewClient(cfg.Agent.SocketPath, cfg.Agent.Token)
	res, err := client.Ping(ctx)
	if err != nil {
		return err
	}
	out, _ := json.MarshalIndent(res, "", "  ")
	fmt.Println(string(out))
	return nil
}

func cmdBootstrapAdmin(ctx context.Context, configPath, username, password string) error {
	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}

	reader := bufio.NewReader(os.Stdin)
	if username == "" {
		fmt.Print("Admin username: ")
		username, _ = reader.ReadString('\n')
		username = strings.TrimSpace(username)
	}
	if username == "" {
		return fmt.Errorf("username cannot be empty")
	}
	if password == "" {
		fmt.Print("Admin password: ")
		password, _ = reader.ReadString('\n')
		password = strings.TrimSpace(password)
	}
	if len(password) < 8 {
		return fmt.Errorf("password must be at least 8 characters")
	}

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	st, err := store.Open(cfg.Database)
	if err != nil {
		return err
	}
	defer func() { _ = st.Close() }()
	if err := st.Migrate(ctx); err != nil {
		return err
	}

	repo := store.NewUserRepo(st.DB)
	count, err := repo.Count(ctx)
	if err != nil {
		return err
	}
	if count > 0 {
		fmt.Printf("Admin user '%s' already exists (count=%d). Skipping.\n", username, count)
		return nil
	}

	hash, err := auth.HashPassword(password)
	if err != nil {
		return err
	}

	u := &core.User{
		Username:     username,
		Email:        username + "@localhost",
		PasswordHash: hash,
		Role:         "admin",
		Language:     "en",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	if err := repo.Create(ctx, u); err != nil {
		return err
	}
	// Ensure signal.NotifyContext import is used.
	_ = syscall.SIGTERM
	fmt.Printf("Admin user '%s' created successfully (id=%d).\n", u.Username, u.ID)
	return nil
}

func reportDoctor(checks []doctorCheck) error {
	failedCritical := false
	for _, c := range checks {
		status := "PASS"
		if !c.OK {
			status = "FAIL"
			if c.Critical {
				failedCritical = true
			}
		}
		fmt.Printf("[%s] %-10s %s\n", status, c.Name, c.Detail)
	}
	if failedCritical {
		return fmt.Errorf("%d critical check(s) failed", countFailed(checks))
	}
	fmt.Println("doctor: all critical checks passed")
	return nil
}

func countFailed(checks []doctorCheck) int {
	n := 0
	for _, c := range checks {
		if !c.OK && c.Critical {
			n++
		}
	}
	return n
}

func statusDetail(err error) string {
	if err == nil {
		return "ok"
	}
	return err.Error()
}
