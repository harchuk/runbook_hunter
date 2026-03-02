package worker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"golang.org/x/sync/errgroup"

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
	cfg, err := s.Settings.EffectiveConfig(ctx)
	if err != nil {
		s.Logger.Error().Err(err).Msg("worker: load effective config failed")
		return
	}
	interval := cfg.Worker.Interval
	if interval <= 0 {
		interval = 30 * time.Second
	}
	ticker := time.NewTicker(interval)
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
	if !ok {
		briefText := fmt.Sprintf("No matching runbook for alert=%s service=%s env=%s", incident.AlertName, incident.Service, incident.Env)
		if err := s.Repo.UpdateIncidentBrief(ctx, incident.ID, briefText, "", incident.Status); err != nil {
			return err
		}
		criteria, closureState, status, streak, executionHash, err := s.evaluateClosure(ctx, cfg, incident, store.RunbookExecutionWithSteps{})
		if err != nil {
			return err
		}
		if err := s.Repo.UpdateIncidentClosure(ctx, incident.ID, criteria, status, closureState, incident.ClosureReason, streak); err != nil {
			return err
		}
		return s.postIncidentUpdate(ctx, cfg, incident, labels, force, destinationIDs, criteria, store.RunbookExecutionWithSteps{}, executionHash, briefText)
	}

	plan := buildExecutionPlan(selected)
	execRow, _, err := s.Repo.EnsureRunbookExecution(ctx, incident.ID, selected.Name, selected.VersionHash, ternary(force, "manual", "auto"), plan)
	if err != nil {
		return err
	}

	steps, err := s.Repo.ListExecutionSteps(ctx, execRow.ID)
	if err != nil {
		return err
	}
	stepByOrder := map[int]runbooks.Step{}
	for i, step := range selected.Steps {
		stepByOrder[i+1] = step
	}

	runner := executor.NewRunner(
		cfg.Security.EgressAllowlist,
		cfg.Limits.StepTimeout,
		cfg.Security.RetryCount,
		executor.NewCircuitBreaker(cfg.Security.CircuitBreakerFails, cfg.Security.CircuitBreakerWindow),
	)

	for _, persisted := range steps {
		spec, exists := stepByOrder[persisted.StepOrder]
		if !exists {
			continue
		}
		if isTerminalStepStatus(persisted.Status) {
			continue
		}

		if strings.EqualFold(spec.Kind, "action") {
			updated, err := s.executeActionStep(ctx, cfg, incident, execRow, persisted, spec)
			if err != nil {
				return err
			}
			if err := s.Repo.UpsertExecutionStep(ctx, execRow.ID, updated); err != nil {
				return err
			}
			continue
		}

		result := runner.RunStep(ctx, spec)
		s.Metrics.StepRunsTotal.WithLabelValues(result.Status).Inc()
		recommendation := spec.RecommendationOnPass
		stepStatus := "ok"
		if result.Status != "ok" {
			stepStatus = "failed"
			recommendation = spec.RecommendationOnFail
		}
		started := result.StartedAt
		finished := result.FinishedAt
		update := store.ExecutionStepInput{
			StepName:       persisted.StepName,
			StepOrder:      persisted.StepOrder,
			Kind:           "check",
			Status:         stepStatus,
			Output:         result.Output,
			Error:          result.Error,
			Recommendation: recommendation,
			Required:       spec.Required,
			StartedAt:      &started,
			FinishedAt:     &finished,
		}
		if err := s.Repo.UpsertExecutionStep(ctx, execRow.ID, update); err != nil {
			return err
		}
		if err := s.Repo.SaveStepRuns(ctx, incident.ID, []store.StepRunInput{{
			StepName:   persisted.StepName,
			Tool:       spec.Tool,
			Status:     result.Status,
			Output:     result.Output,
			Error:      result.Error,
			StartedAt:  result.StartedAt,
			FinishedAt: result.FinishedAt,
		}}); err != nil {
			return err
		}
	}

	execution, err := s.Repo.GetLatestRunbookExecution(ctx, incident.ID)
	if err != nil {
		return err
	}
	execStatus, active := deriveExecutionStatus(execution.Steps)
	if err := s.Repo.UpdateExecutionStatus(ctx, execution.Execution.ID, execStatus, active); err != nil {
		return err
	}
	execution.Execution.Status = execStatus
	execution.Execution.Active = active

	briefText := buildExecutionBrief(incident, execution)
	if err := s.Repo.UpdateIncidentBrief(ctx, incident.ID, briefText, selected.Name, incident.Status); err != nil {
		return err
	}

	criteria, closureState, status, streak, executionHash, err := s.evaluateClosure(ctx, cfg, incident, execution)
	if err != nil {
		return err
	}
	if err := s.Repo.UpdateIncidentClosure(ctx, incident.ID, criteria, status, closureState, deriveClosureReason(criteria, closureState, incident), streak); err != nil {
		return err
	}
	return s.postIncidentUpdate(ctx, cfg, incident, labels, force, destinationIDs, criteria, execution, executionHash, briefText)
}

func buildExecutionPlan(book runbooks.Definition) []store.ExecutionStepInput {
	plan := make([]store.ExecutionStepInput, 0, len(book.Steps))
	for i, step := range book.Steps {
		kind := step.Kind
		if strings.TrimSpace(kind) == "" {
			kind = "check"
		}
		plan = append(plan, store.ExecutionStepInput{
			StepName:       step.Name,
			StepOrder:      i + 1,
			Kind:           kind,
			Status:         "pending",
			Recommendation: "",
			Required:       step.Required,
		})
	}
	return plan
}

func isTerminalStepStatus(status string) bool {
	s := strings.ToLower(strings.TrimSpace(status))
	switch s {
	case "ok", "failed", "action_failed_blocking", "skipped", "rejected":
		return true
	default:
		return false
	}
}

func deriveExecutionStatus(steps []store.RunbookExecutionStep) (string, bool) {
	if len(steps) == 0 {
		return "running", true
	}
	hasPending := false
	hasBlocking := false
	hasFailed := false
	for _, step := range steps {
		s := strings.ToLower(strings.TrimSpace(step.Status))
		switch s {
		case "pending", "running", "pending_approval", "approved":
			hasPending = true
		case "action_failed_blocking":
			hasBlocking = true
		case "failed":
			if step.Required {
				hasBlocking = true
			}
			hasFailed = true
		}
	}
	if hasPending {
		return "running", true
	}
	if hasBlocking {
		return "blocked", true
	}
	if hasFailed {
		return "completed_with_errors", false
	}
	return "completed", false
}

func buildExecutionBrief(incident store.Incident, execution store.RunbookExecutionWithSteps) string {
	if len(execution.Steps) == 0 {
		return fmt.Sprintf("No step results yet for incident %d", incident.ID)
	}
	sorted := append([]store.RunbookExecutionStep(nil), execution.Steps...)
	sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].StepOrder < sorted[j].StepOrder })
	parts := make([]string, 0, len(sorted)+1)
	okCount := 0
	failCount := 0
	for _, step := range sorted {
		status := strings.ToLower(strings.TrimSpace(step.Status))
		switch status {
		case "ok":
			okCount++
			parts = append(parts, fmt.Sprintf("%s: ok", step.StepName))
		case "failed", "action_failed_blocking", "rejected":
			failCount++
			detail := strings.TrimSpace(step.Error)
			if detail == "" {
				detail = status
			}
			parts = append(parts, fmt.Sprintf("%s: %s", step.StepName, detail))
		default:
			parts = append(parts, fmt.Sprintf("%s: %s", step.StepName, status))
		}
	}
	return fmt.Sprintf("%s (%s/%s): %d ok, %d failed. %s", incident.AlertName, incident.Service, incident.Env, okCount, failCount, strings.Join(parts, "; "))
}

func deriveClosureReason(criteria store.ClosureCriteriaSnapshot, closureState string, incident store.Incident) string {
	if closureState == "closed" && criteria.ClosureReady {
		return "auto:criteria_met"
	}
	if strings.TrimSpace(incident.ClosureReason) != "" {
		return incident.ClosureReason
	}
	return ""
}

func (s *Service) evaluateClosure(ctx context.Context, cfg config.Config, incident store.Incident, execution store.RunbookExecutionWithSteps) (store.ClosureCriteriaSnapshot, string, string, int, string, error) {
	_, firing, err := s.Repo.IncidentAlertStats(ctx, incident.ID)
	if err != nil {
		return store.ClosureCriteriaSnapshot{}, "", "", 0, "", err
	}
	alertsResolved := firing == 0
	requiredStepsPassed := true
	noBlockers := true
	if len(execution.Steps) > 0 {
		for _, step := range execution.Steps {
			status := strings.ToLower(strings.TrimSpace(step.Status))
			if step.Required && status != "ok" {
				requiredStepsPassed = false
			}
			if status == "action_failed_blocking" || status == "rejected" {
				noBlockers = false
			}
		}
	}
	closureReady := alertsResolved && requiredStepsPassed && noBlockers
	streak := incident.ClosureReadyStreak
	if closureReady {
		streak++
	} else {
		streak = 0
	}

	closureState := incident.ClosureState
	if closureState == "" {
		closureState = "open"
	}
	status := "open"
	if closureReady {
		closureState = "ready_to-close"
	}
	if strings.EqualFold(cfg.Closure.Policy, "auto+manual") && closureReady && streak >= cfg.Closure.AutoCloseConsecutive {
		closureState = "closed"
		status = "resolved"
	}
	if !closureReady && closureState == "ready_to-close" {
		closureState = "open"
	}

	snapshot := store.ClosureCriteriaSnapshot{
		AlertsResolved:      alertsResolved,
		RequiredStepsPassed: requiredStepsPassed,
		NoBlockers:          noBlockers,
		ClosureReady:        closureReady,
		LastEvaluatedAt:     time.Now().UTC(),
	}
	executionHash := store.HashContent(fmt.Sprintf("incident=%d|firing=%d|ready=%t|exec=%s|steps=%s", incident.ID, firing, closureReady, execution.Execution.Status, summarizeExecutionSteps(execution.Steps)))
	return snapshot, closureState, status, streak, executionHash, nil
}

func summarizeExecutionSteps(steps []store.RunbookExecutionStep) string {
	if len(steps) == 0 {
		return "no-steps"
	}
	items := make([]string, 0, len(steps))
	for _, step := range steps {
		items = append(items, fmt.Sprintf("%d:%s:%s", step.StepOrder, step.StepName, step.Status))
	}
	sort.Strings(items)
	return strings.Join(items, "|")
}

func (s *Service) postIncidentUpdate(ctx context.Context, cfg config.Config, incident store.Incident, labels map[string]string, force bool, destinationIDs []string, criteria store.ClosureCriteriaSnapshot, execution store.RunbookExecutionWithSteps, executionHash, briefText string) error {
	destinations := s.Router.SelectDestinations(labels, cfg, destinationIDs)
	for _, destination := range destinations {
		message := s.formatMessage(destination, incident, briefText, execution, criteria)
		hash := store.HashContent(message)
		shouldSend, err := s.Repo.CheckAndUpdateDedupWithExecution(ctx, incident.ID, destination.ID, hash, executionHash, cfg.Limits.DedupCooldown, force, time.Now().UTC())
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
			continue
		}
		_ = s.Repo.RecordIncidentEvent(ctx, incident.ID, "notification_sent", "incident update sent", map[string]any{"destination_id": destination.ID, "destination_type": destination.Type})
	}
	return nil
}

func (s *Service) formatMessage(destination router.ResolvedDestination, incident store.Incident, briefText string, execution store.RunbookExecutionWithSteps, criteria store.ClosureCriteriaSnapshot) string {
	runbookName := incident.RunbookName
	if runbookName == "" {
		runbookName = execution.Execution.RunbookName
	}
	data := reporter.MessageData{
		Severity:       nonEmpty(incident.Severity, "unknown"),
		AlertName:      nonEmpty(incident.AlertName, "unknown"),
		Service:        nonEmpty(incident.Service, "unknown"),
		Env:            nonEmpty(incident.Env, "unknown"),
		IncidentID:     incident.ID,
		Status:         nonEmpty(incident.Status, "open"),
		Brief:          briefText,
		RunbookName:    nonEmpty(runbookName, "n/a"),
		UpdatedAt:      time.Now().UTC(),
		ClosureState:   nonEmpty(incident.ClosureState, "open"),
		ClosureSummary: fmt.Sprintf("alerts_resolved=%t, required_steps_passed=%t, no_blockers=%t", criteria.AlertsResolved, criteria.RequiredStepsPassed, criteria.NoBlockers),
		StepSummary:    summarizeExecutionSteps(execution.Steps),
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

func ternary(flag bool, whenTrue, whenFalse string) string {
	if flag {
		return whenTrue
	}
	return whenFalse
}

func (s *Service) executeActionStep(ctx context.Context, cfg config.Config, incident store.Incident, execution store.RunbookExecution, persisted store.RunbookExecutionStep, spec runbooks.Step) (store.ExecutionStepInput, error) {
	now := time.Now().UTC()
	out := store.ExecutionStepInput{
		StepName:   persisted.StepName,
		StepOrder:  persisted.StepOrder,
		Kind:       "action",
		Required:   spec.Required,
		StartedAt:  &now,
		FinishedAt: nil,
	}

	if !strings.EqualFold(spec.Tool, "ansible_awx_job") {
		finish := time.Now().UTC()
		out.Status = "action_failed_blocking"
		out.Error = "unsupported action tool"
		out.Recommendation = nonEmpty(spec.RecommendationOnFail, "Remove unsupported action step from runbook.")
		out.FinishedAt = &finish
		return out, nil
	}

	if spec.RequiresApproval {
		req, err := s.Repo.GetOrCreateApproval(ctx, incident.ID, execution.ID, persisted.StepOrder, "ansible_awx_job", "AWX action requires approval")
		if err != nil {
			return out, err
		}
		switch req.Status {
		case "pending":
			out.Status = "pending_approval"
			out.Output = fmt.Sprintf("approval request #%d pending", req.ID)
			return out, nil
		case "rejected":
			finish := time.Now().UTC()
			out.Status = "action_failed_blocking"
			out.Error = "approval rejected"
			out.Recommendation = nonEmpty(spec.RecommendationOnFail, "Review risk and resubmit approval request.")
			out.FinishedAt = &finish
			return out, nil
		}
	}

	launchID, payload, err := triggerAWXJob(ctx, cfg, spec)
	finish := time.Now().UTC()
	if err != nil {
		out.Status = "action_failed_blocking"
		out.Error = err.Error()
		out.Output = payload
		out.Recommendation = nonEmpty(spec.RecommendationOnFail, "Check AWX connectivity, template permissions and credentials.")
		out.FinishedAt = &finish
		return out, nil
	}
	out.Status = "ok"
	out.Output = fmt.Sprintf("awx job launched: %d", launchID)
	out.Recommendation = nonEmpty(spec.RecommendationOnPass, "Track AWX job and verify remediation impact.")
	out.FinishedAt = &finish
	_ = s.Repo.RecordIncidentEvent(ctx, incident.ID, "ansible_triggered", "awx job launched", map[string]any{"execution_id": execution.ID, "step_order": persisted.StepOrder, "job_id": launchID})
	return out, nil
}

func triggerAWXJob(ctx context.Context, cfg config.Config, step runbooks.Step) (int64, string, error) {
	if !cfg.AWX.Enabled {
		return 0, "", fmt.Errorf("awx integration is disabled")
	}
	if strings.TrimSpace(cfg.AWX.URL) == "" {
		return 0, "", fmt.Errorf("awx url is not configured")
	}
	if strings.TrimSpace(cfg.AWX.Token) == "" {
		return 0, "", fmt.Errorf("awx token is not configured")
	}
	templateID := step.AWXTemplateID
	if templateID <= 0 {
		return 0, "", fmt.Errorf("awx_template_id is required")
	}
	inventoryID := step.InventoryID
	if inventoryID <= 0 {
		inventoryID = cfg.AWX.DefaultInventoryID
	}

	body := map[string]any{
		"extra_vars": step.ExtraVars,
	}
	if inventoryID > 0 {
		body["inventory"] = inventoryID
	}
	if strings.TrimSpace(step.PlaybookRef) != "" {
		body["playbook_ref"] = step.PlaybookRef
	}
	blob, _ := json.Marshal(body)

	reqCtx, cancel := context.WithTimeout(ctx, cfg.AWX.RequestTimeout)
	defer cancel()
	url := fmt.Sprintf("%s/api/v2/job_templates/%d/launch/", strings.TrimRight(cfg.AWX.URL, "/"), templateID)
	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, url, bytes.NewReader(blob))
	if err != nil {
		return 0, "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+cfg.AWX.Token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, "", err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 8192))
	if resp.StatusCode >= 300 {
		return 0, string(respBody), fmt.Errorf("awx launch failed status=%d", resp.StatusCode)
	}
	var decoded struct {
		ID int64 `json:"id"`
	}
	if err := json.Unmarshal(respBody, &decoded); err != nil {
		return 0, string(respBody), nil
	}
	return decoded.ID, string(respBody), nil
}
