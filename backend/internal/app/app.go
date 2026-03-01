package app

import (
	"fmt"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog"

	"github.com/runbook-hunter/runbook-hunter/backend/internal/config"
	"github.com/runbook-hunter/runbook-hunter/backend/internal/obs"
	"github.com/runbook-hunter/runbook-hunter/backend/internal/reporter"
	"github.com/runbook-hunter/runbook-hunter/backend/internal/router"
	"github.com/runbook-hunter/runbook-hunter/backend/internal/settings"
	"github.com/runbook-hunter/runbook-hunter/backend/internal/store"
	"github.com/runbook-hunter/runbook-hunter/backend/internal/worker"
)

type App struct {
	Config   config.Config
	Repo     *store.Repository
	Settings *settings.Service
	Router   *router.Engine
	Metrics  *obs.Metrics
	Logger   zerolog.Logger
	Worker   *worker.Service
}

func New(cfg config.Config, logger zerolog.Logger) (*App, error) {
	repo, err := store.Open(cfg.Database.Driver, cfg.Database.DSN)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	if cfg.Database.Driver == "sqlite" {
		if err := repo.EnsureLocalSchema(); err != nil {
			return nil, fmt.Errorf("ensure sqlite schema: %w", err)
		}
	}

	settingsSvc, err := settings.NewService(repo, cfg, cfg.Security.SettingsCryptoKeyB64)
	if err != nil {
		return nil, err
	}

	reg := prometheus.DefaultRegisterer
	metrics := obs.NewMetrics(reg)
	routerEngine := router.New()

	workerSvc := &worker.Service{
		Repo:       repo,
		Settings:   settingsSvc,
		Router:     routerEngine,
		Metrics:    metrics,
		Logger:     logger,
		Telegram:   reporter.NewTelegramReporter(cfg.Security.RequestTimeout),
		Mattermost: reporter.NewMattermostReporter(cfg.Security.RequestTimeout),
	}

	return &App{
		Config:   cfg,
		Repo:     repo,
		Settings: settingsSvc,
		Router:   routerEngine,
		Metrics:  metrics,
		Logger:   logger,
		Worker:   workerSvc,
	}, nil
}
