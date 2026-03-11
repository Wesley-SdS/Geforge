package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/wesleybatista/gateforge/internal/config"
	"github.com/wesleybatista/gateforge/internal/observability"
	"github.com/wesleybatista/gateforge/internal/server"

	"log/slog"
)

// Version is set at build time via ldflags.
var Version = "dev"

func main() {
	configPath := flag.String("config", "configs/gateway.yaml", "path to config file")
	version := flag.Bool("version", false, "print version and exit")
	flag.Parse()

	if *version {
		fmt.Printf("gateforge %s\n", Version)
		os.Exit(0)
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		os.Exit(1)
	}

	logger := observability.NewLogger(cfg.Logging)
	logger.Info("gateforge starting",
		slog.String("version", Version),
		slog.Int("port", cfg.Server.Port),
		slog.Int("routes", len(cfg.Routes)),
	)

	// Context for background goroutines (health checkers)
	appCtx, appCancel := context.WithCancel(context.Background())
	defer appCancel()

	handler := server.BuildRouter(appCtx, cfg, logger)

	srv := server.New(
		handler,
		cfg.Server.Port,
		cfg.Server.ReadTimeout,
		cfg.Server.WriteTimeout,
		cfg.Server.IdleTimeout,
		logger,
	)

	// Start config watcher for hot reload
	watcher, err := config.NewWatcher(*configPath, func(newCfg *config.GatewayConfig) {
		logger.Info("config reloaded, rebuilding router")
		newHandler := server.BuildRouter(appCtx, newCfg, logger)
		srv.SetHandler(newHandler)
	}, logger)
	if err != nil {
		logger.Warn("config watcher failed, hot reload disabled", slog.String("error", err.Error()))
	} else {
		defer watcher.Close()
	}

	// Graceful shutdown
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)
		sig := <-sigCh
		logger.Info("shutdown signal received", slog.String("signal", sig.String()))

		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		if err := srv.Shutdown(ctx); err != nil {
			logger.Error("server shutdown error", slog.String("error", err.Error()))
		}
	}()

	if err := srv.Start(); err != nil {
		logger.Error("server failed", slog.String("error", err.Error()))
		os.Exit(1)
	}
	logger.Info("server stopped gracefully")
}
