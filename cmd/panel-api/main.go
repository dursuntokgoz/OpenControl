// Command panel-api is the public HTTP API server of ServerPanel.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/dursuntokgoz/OpenControl/internal/agent"
	"github.com/dursuntokgoz/OpenControl/internal/api"
	"github.com/dursuntokgoz/OpenControl/internal/auth"
	"github.com/dursuntokgoz/OpenControl/internal/config"
	"github.com/dursuntokgoz/OpenControl/internal/core"
	"github.com/dursuntokgoz/OpenControl/internal/store"
)

func main() {
	configPath := flag.String("config", os.Getenv("SERVERPANEL_CONFIG"), "path to config.yaml")
	flag.Parse()

	logger := newLogger()
	info := core.CurrentBuild()
	logger.Info("starting", "component", "panel-api", "version", info.Version, "commit", info.Commit)

	cfg, err := config.Load(*configPath)
	if err != nil {
		logger.Error("load config", slog.Any("err", err))
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	st, err := store.Open(cfg.Database)
	if err != nil {
		logger.Error("open store", slog.Any("err", err))
		os.Exit(1)
	}
	defer func() {
		if closeErr := st.Close(); closeErr != nil {
			logger.Error("close store", slog.Any("err", closeErr))
		}
	}()
	if err := st.Migrate(ctx); err != nil {
		logger.Error("migrate", slog.Any("err", err))
		os.Exit(1)
	}

	bootstrapIfConfigured(ctx, st, logger)

	userRepo := store.NewUserRepo(st.DB)
	auditRepo := store.NewAuditRepo(st.DB)
	sessionRepo := store.NewSessionStore(st.DB)
	pkgRepo := store.NewPackageRepo(st.DB)
	acctRepo := store.NewAccountRepo(st.DB)
	sessMgr := auth.NewSessionManager(sessionRepo, 12*time.Hour)

	_ = agent.NewClient(cfg.Agent.SocketPath, cfg.Agent.Token)

	router := api.NewRouter(api.Options{
		WebDist:  webDistOrDefault(cfg.HTTP.WebDist),
		Logger:   logger,
		Sessions: sessMgr,
		Deps: &core.AppDeps{
			Users:    userRepo,
			Audit:    auditRepo,
			Packages: pkgRepo,
			Accounts: acctRepo,
		},
	})

	srv := &http.Server{
		Addr:              cfg.HTTP.Listen,
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       60 * time.Second,
		WriteTimeout:      120 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		logger.Info("listening", "addr", cfg.HTTP.Listen)
		errCh <- srv.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			logger.Error("shutdown", slog.Any("err", err))
		}
	case err := <-errCh:
		if !errors.Is(err, http.ErrServerClosed) {
			logger.Error("server error", slog.Any("err", err))
			stop()
			os.Exit(1)
		}
	}
	logger.Info("stopped")
}

// bootstrapIfConfigured auto-creates an admin user when SERVERPANEL_BOOTSTRAP_ADMIN_USER
// and SERVERPANEL_BOOTSTRAP_ADMIN_PASSWORD are set and no users exist.
func bootstrapIfConfigured(ctx context.Context, st *store.DB, logger *slog.Logger) {
	username := os.Getenv("SERVERPANEL_BOOTSTRAP_ADMIN_USER")
	password := os.Getenv("SERVERPANEL_BOOTSTRAP_ADMIN_PASSWORD")
	if username == "" || password == "" {
		return
	}
	repo := store.NewUserRepo(st.DB)
	count, err := repo.Count(ctx)
	if err != nil {
		logger.Warn("bootstrap: count users", slog.Any("err", err))
		return
	}
	if count > 0 {
		return
	}
	hash, err := auth.HashPassword(password)
	if err != nil {
		logger.Error("bootstrap: hash password", slog.Any("err", err))
		return
	}
	email := os.Getenv("SERVERPANEL_BOOTSTRAP_ADMIN_EMAIL")
	if email == "" {
		email = username + "@localhost"
	}
	u := &core.User{
		Username:     strings.TrimSpace(username),
		Email:        strings.TrimSpace(email),
		PasswordHash: hash,
		Role:         "admin",
		Language:     "en",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	if err := repo.Create(ctx, u); err != nil {
		logger.Error("bootstrap: create admin", slog.Any("err", err))
		return
	}
	logger.Info("bootstrap: admin user created", "username", u.Username)
}

func newLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
}

func webDistOrDefault(dist string) string {
	if dist == "" {
		return ""
	}
	if fi, err := os.Stat(filepath.Join(dist, "index.html")); err == nil && !fi.IsDir() {
		return dist
	}
	return ""
}

func init() {
	_ = fmt.Sprintf // keep fmt imported
}
