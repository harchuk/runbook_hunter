package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"golang.org/x/sync/errgroup"

	"github.com/runbook-hunter/runbook-hunter/backend/internal/brief"
	"github.com/runbook-hunter/runbook-hunter/backend/internal/config"
	"github.com/runbook-hunter/runbook-hunter/backend/internal/executor"
	"github.com/runbook-hunter/runbook-hunter/backend/internal/obs"
	"github.com/runbook-hunter/runbook-hunter/backend/internal/reporter"
	"github.com/runbook-hunter/runbook-hunter/backend/internal/router"
	"github.com/runbook-hunter/runbook-hunter/backend/internal/runbooks"
	"github.com/runbook-hunter/runbook-hunter/backend/internal/settings"
	"github.com/runbook-hunter/runbook-hunter/backend/internal/store"
)

type Service struct {
	Repo       *store.Repository
	Settings   *settings.Service
	Router     *router.Engine
	Metrics    *obs.Metrics
	Logger     zerolog.Logger
	Telegram   *reporter.TelegramReporter
	Mattermost *reporter.MattermostReporter
}

func (s *Service) Run(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	s.runOnce(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.runOnce(ctx)
		}
	}
}

func (s *Service) runOnce(ctx context.Context) {
	cfg, err := s.Settings.EffectiveConfig(ctx)
	if err != nil {
		s.Logger.Error().Err(err).Msg("worker: load effective config failed")
		s.Metrics.ErrorsTotal.WithLabelValues("worker_settings").Inc()
		return
	}

	incidents, err := s.Repo.GetOpenIncidents(ctx, cfg.Worker.BatchSize)
	if err != nil {
		s.Logger.Error().Err(err).Msg("worker: fetch open incidents failed")
		s.Metrics.ErrorsTotal.WithLabelValues("worker_open_incidents").Inc()
		return
	}
	s.Metrics.IncidentsOpen.Set(float64(len(incidents)))

	books, err := runbooks.Loader{Mode: cfg.Runbooks.Mode, Path: cfg.Runbooks.Path, Store: s.Repo}.Load(ctx)
	if err != nil {
		s.Logger.Error().Err(err).Msg("worker: load runbooks failed")
		s.Metrics.ErrorsTotal.WithLabelValues("worker_runbooks").Inc()
		return
	}

	g, childCtx := errgroup.WithContext(ctx)
	g.SetLimit(cfg.Worker.MaxConcurrency)
	for _, incident := range incidents {
		incident := incident
		g.Go(func() error {
			return s.processIncident(childCtx, cfg, books, incident, false, nil)
		})
	}
	if err := g.Wait(); err != nil {
		s.Logger.Error().Err(err).Msg("worker: incident processing finished with errors")
	}
}

func (s *Service) PostUpdateNow(ctx context.Context, incidentID uint, destinationIDs []string) error {
	cfg, err := s.Settings.EffectiveConfig(ctx)
	if err != nil {
		return err
	}
	books, err := runbooks.Loader{Mode: cfg.Runbooks.Mode, Path: cfg.Runbooks.Path, Store: s.Repo}.Load(ctx)
	if err != nil {
		return err
	}
	incident, err := s.Repo.GetIncidentByID(ctx, incidentID)
	if err != nil {
		return err
	}
	return s.processIncident(ctx, cfg, books, incident, true, destinationIDs)
}

func (s *Service) processIncident(ctx context.Context, cfg config.Config, books []runbooks.Definition, incident store.Incident, force bool, destinationIDs []string) error {
	labels := map[string]string{}
	_ = json.Unmarshal(incident.Labels, &labels)

	selected, ok := runbooks.Match(labels, books)
	runbookName := ""
	results := []executor.Result{}
	if ok {
		runbookName = selected.Name
		runner := executor.NewRunner(
			cfg.Security.EgressAllowlist,
			cfg.Limits.StepTimeout,
			cfg.Security.RetryCount,
			executor.NewCircuitBreaker(cfg.Security.CircuitBreakerFails, cfg.Security.CircuitBreakerWindow),
		)
		results = runner.Run(ctx, selected.Steps, cfg.Limits.MaxSteps)
	}

	stepRuns := make([]store.StepRunInput, 0, len(results))
	for _, result := range results {
		stepRuns = append(stepRuns, store.StepRunInput{
			StepName:   result.StepName,
			Tool:       result.Tool,
			Status:     result.Status,
			Output:     result.Output,
			Error:      result.Error,
			StartedAt:  result.StartedAt,
			FinishedAt: result.FinishedAt,
		})
		s.Metrics.StepRunsTotal.WithLabelValues(result.Status).Inc()
	}
	if err := s.Repo.SaveStepRuns(ctx, incident.ID, stepRuns); err != nil {
		s.Metrics.ErrorsTotal.WithLabelValues("worker_save_steps").Inc()
		return err
	}

	briefText := brief.Build(incident.AlertName, incident.Service, incident.Env, results)
	if err := s.Repo.UpdateIncidentBrief(ctx, incident.ID, briefText, runbookName, incident.Status); err != nil {
		s.Metrics.ErrorsTotal.WithLabelValues("worker_update_incident").Inc()
		return err
	}

	destinations := s.Router.SelectDestinations(labels, cfg, destinationIDs)
	for _, destination := range destinations {
		message := s.formatMessage(destination, incident, briefText, runbookName)
		hash := store.HashContent(message)
		shouldSend, err := s.Repo.CheckAndUpdateDedup(ctx, incident.ID, destination.ID, hash, cfg.Limits.DedupCooldown, force, time.Now().UTC())
		if err != nil {
			s.Metrics.ErrorsTotal.WithLabelValues("worker_dedup").Inc()
			return err
		}
		if !shouldSend {
			continue
		}
		if err := s.send(ctx, destination, message); err != nil {
			s.Metrics.ErrorsTotal.WithLabelValues("worker_send").Inc()
			s.Logger.Error().Err(err).Str("destination", destination.ID).Msg("worker send failed")
		}
	}
	return nil
}

func (s *Service) formatMessage(destination router.ResolvedDestination, incident store.Incident, briefText, runbookName string) string {
	data := reporter.MessageData{
		Severity:    nonEmpty(incident.Severity, "unknown"),
		AlertName:   nonEmpty(incident.AlertName, "unknown"),
		Service:     nonEmpty(incident.Service, "unknown"),
		Env:         nonEmpty(incident.Env, "unknown"),
		IncidentID:  incident.ID,
		Status:      nonEmpty(incident.Status, "open"),
		Brief:       briefText,
		RunbookName: nonEmpty(runbookName, "n/a"),
		UpdatedAt:   time.Now().UTC(),
	}
	if destination.Type == "mattermost" {
		return reporter.FormatMattermostMessage(data)
	}
	return reporter.FormatTelegramMessage(data)
}

func (s *Service) send(ctx context.Context, destination router.ResolvedDestination, message string) error {
	switch destination.Type {
	case "telegram":
		if destination.Telegram == nil {
			return fmt.Errorf("telegram destination is nil")
		}
		if err := s.Telegram.Send(ctx, *destination.Telegram, message); err != nil {
			return err
		}
		s.Metrics.TelegramSent.Inc()
	case "mattermost":
		if destination.Mattermost == nil {
			return fmt.Errorf("mattermost destination is nil")
		}
		if err := s.Mattermost.Send(ctx, *destination.Mattermost, message); err != nil {
			return err
		}
		s.Metrics.MattermostSent.Inc()
	default:
		return fmt.Errorf("unsupported destination type %s", destination.Type)
	}
	return nil
}

func nonEmpty(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
