// Command panel-api is the public HTTP API server of ServerPanel. It runs as
// an unprivileged user and never touches the system directly; privileged work
// is delegated to panel-agent over a Unix domain socket.
package main

import (
	"context"
	"errors"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/dursuntokgoz/OpenControl/internal/agent"
	"github.com/dursuntokgoz/OpenControl/internal/api"
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

	agentClient := agent.NewClient(cfg.Agent.SocketPath, cfg.Agent.Token)
	_ = agentClient // used by handlers from phase 2 onward; dial is lazy.

	router := api.NewRouter(api.Options{
		WebDist: webDistOrDefault(cfg.HTTP.WebDist),
		Logger:  logger,
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

func newLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
}

// webDistOrDefault returns "" when the dist directory does not exist so the
// API can run head-less (tests, CLI-only hosts).
func webDistOrDefault(dist string) string {
	if dist == "" {
		return ""
	}
	if fi, err := os.Stat(filepath.Join(dist, "index.html")); err == nil && !fi.IsDir() {
		return dist
	}
	return ""
}
