// Command panel-agent is the privileged helper process. It runs as root,
// listens on a Unix domain socket and executes only whitelisted operations.
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/dursuntokgoz/OpenControl/internal/agent"
	"github.com/dursuntokgoz/OpenControl/internal/config"
	"github.com/dursuntokgoz/OpenControl/internal/core"
)

func main() {
	configPath := flag.String("config", os.Getenv("SERVERPANEL_CONFIG"), "path to config.yaml")
	socketOverride := flag.String("socket", "", "override agent socket path")
	tokenOverride := flag.String("token", "", "override agent token (prefer env)")
	flag.Parse()

	logger := newLogger()
	info := core.CurrentBuild()
	logger.Info("starting", "component", "panel-agent", "version", info.Version)

	cfg, err := config.Load(*configPath)
	if err != nil {
		logger.Error("load config", slog.Any("err", err))
		os.Exit(1)
	}
	socketPath := cfg.Agent.SocketPath
	if *socketOverride != "" {
		socketPath = *socketOverride
	}
	token := cfg.Agent.Token
	if *tokenOverride != "" {
		token = *tokenOverride
	}
	if token == "" {
		token = os.Getenv("SERVERPANEL_AGENT_TOKEN")
	}

	srv := agent.NewServer(socketPath, token, logger)
	agent.RegisterBuiltinOps(srv)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := srv.Listen(); err != nil {
		logger.Error("listen", slog.String("socket", socketPath), slog.Any("err", err))
		os.Exit(1)
	}
	logger.Info("agent listening", "socket", socketPath)
	if err := srv.Serve(ctx); err != nil {
		logger.Error("serve", slog.Any("err", fmt.Errorf("%w", err)))
		os.Exit(1)
	}
	logger.Info("agent stopped")
}

func newLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
}
