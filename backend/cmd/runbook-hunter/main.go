package main

import (
	"context"
	"errors"
	"flag"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/runbook-hunter/runbook-hunter/backend/internal/app"
	"github.com/runbook-hunter/runbook-hunter/backend/internal/config"
	httpserver "github.com/runbook-hunter/runbook-hunter/backend/internal/http"
	"github.com/runbook-hunter/runbook-hunter/backend/internal/obs"
	"github.com/runbook-hunter/runbook-hunter/backend/internal/store"
)

func main() {
	var (
		mode           = flag.String("mode", "api", "run mode: api or worker")
		configPath     = flag.String("config", "", "path to config yaml")
		runMigrations  = flag.Bool("run-migrations", true, "run database migrations before start")
		migrationsPath = flag.String("migrations-path", "internal/store/migrations", "path to migrations")
		logLevel       = flag.String("log-level", "info", "log level")
	)
	flag.Parse()

	logger := obs.NewLogger(*logLevel, os.Stdout)
	cfg, err := config.Load(*configPath)
	if err != nil {
		logger.Fatal().Err(err).Msg("load config failed")
	}

	if *runMigrations {
		if err := store.RunMigrations(cfg.Database.Driver, cfg.Database.DSN, *migrationsPath); err != nil {
			logger.Fatal().Err(err).Msg("run migrations failed")
		}
	}

	application, err := app.New(cfg, logger)
	if err != nil {
		logger.Fatal().Err(err).Msg("init app failed")
	}
	defer func() { _ = application.Repo.Close() }()

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	switch *mode {
	case "api":
		server := httpserver.NewServer(application)
		go func() {
			<-ctx.Done()
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			if err := server.Shutdown(shutdownCtx); err != nil {
				application.Logger.Error().Err(err).Msg("api shutdown failed")
			}
		}()
		if err := server.Start(); err != nil && !errors.Is(err, context.Canceled) && !errors.Is(err, syscall.EINVAL) && err.Error() != "http: Server closed" {
			application.Logger.Fatal().Err(err).Msg("api exited with error")
		}
	case "worker":
		application.Worker.Run(ctx)
	default:
		application.Logger.Fatal().Str("mode", *mode).Msg("unsupported mode")
	}
}
