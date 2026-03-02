package store

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type AlertSignalInput struct {
	Fingerprint  string
	RouteKey     string
	AlertName    string
	Status       string
	Labels       map[string]string
	Annotations  map[string]string
	StartsAt     time.Time
	EndsAt       *time.Time
	GeneratorURL string
}

type StepRunInput struct {
	StepName   string
	Tool       string
	Status     string
	Output     string
	Error      string
	StartedAt  time.Time
	FinishedAt time.Time
}

type ExecutionStepInput struct {
	StepName       string
	StepOrder      int
	Kind           string
	Status         string
	Output         string
	Error          string
	Recommendation string
	Required       bool
	StartedAt      *time.Time
	FinishedAt     *time.Time
}

type IncidentWithSteps struct {
	Incident   Incident                    `json:"incident"`
	Steps      []StepRun                   `json:"steps"`
	Executions []RunbookExecutionWithSteps `json:"executions"`
	Alerts     []Signal                    `json:"alerts"`
}

type RunbookExecutionWithSteps struct {
	Execution RunbookExecution       `json:"execution"`
	Steps     []RunbookExecutionStep `json:"steps"`
}

type IncidentListItem struct {
	Incident
	AlertsTotal  int  `json:"alerts_total"`
	AlertsFiring int  `json:"alerts_firing"`
	ClosureReady bool `json:"closure_ready"`
}

type ClosureCriteriaSnapshot struct {
	AlertsResolved      bool      `json:"alerts_resolved"`
	RequiredStepsPassed bool      `json:"required_steps_passed"`
	NoBlockers          bool      `json:"no_blockers"`
	ClosureReady        bool      `json:"closure_ready"`
	LastEvaluatedAt     time.Time `json:"last_evaluated_at"`
}

type GitOpsChangeInput struct {
	ChangeType string
	Title      string
	Desired    map[string]any
}

func (r *Repository) UpsertIncidentAndSignal(ctx context.Context, in AlertSignalInput) (Incident, error) {
	labelsJSON, err := json.Marshal(in.Labels)
	if err != nil {
		return Incident{}, err
	}
	annotationsJSON, err := json.Marshal(in.Annotations)
	if err != nil {
		return Incident{}, err
	}

	var incident Incident
	now := time.Now().UTC()
	err = r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		err := tx.Where("fingerprint = ?", in.Fingerprint).First(&incident).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			incident = Incident{
				Fingerprint:        in.Fingerprint,
				RouteKey:           in.RouteKey,
				AlertName:          in.AlertName,
				Service:            in.Labels["service"],
				Env:                in.Labels["env"],
				Severity:           in.Labels["severity"],
				Status:             normalizeIncidentStatus(in.Status),
				OpenedAt:           in.StartsAt,
				LastSignalAt:       now,
				Labels:             labelsJSON,
				ClosureState:       "open",
				ClosureReason:      "",
				ClosureCriteria:    []byte("{}"),
				ResolvedAt:         nil,
				ClosureReadyStreak: 0,
			}
			if normalizeIncidentStatus(in.Status) == "resolved" {
				incident.ClosureState = "ready_to-close"
			}
			if err := tx.Create(&incident).Error; err != nil {
				return err
			}
		} else if err != nil {
			return err
		} else {
			closureState := incident.ClosureState
			closureReason := incident.ClosureReason
			updates := map[string]any{
				"route_key":            in.RouteKey,
				"severity":             in.Labels["severity"],
				"status":               normalizeIncidentStatus(in.Status),
				"last_signal_at":       now,
				"labels":               labelsJSON,
				"service":              in.Labels["service"],
				"env":                  in.Labels["env"],
				"closure_reason":       closureReason,
				"closure_state":        closureState,
				"closure_ready_streak": 0,
			}
			if strings.EqualFold(in.Status, "firing") {
				updates["status"] = "open"
				updates["closed_at"] = nil
				updates["resolved_at"] = nil
				if incident.ClosureState == "closed" {
					updates["closure_state"] = "reopened"
					updates["closure_reason"] = "auto:reopened_by_signal"
				}
			}
			if err := tx.Model(&Incident{}).Where("id = ?", incident.ID).Updates(updates).Error; err != nil {
				return err
			}
			if err := tx.Where("id = ?", incident.ID).First(&incident).Error; err != nil {
				return err
			}
		}

		signal := Signal{
			IncidentID:   incident.ID,
			Fingerprint:  in.Fingerprint,
			AlertName:    in.AlertName,
			Status:       in.Status,
			Labels:       labelsJSON,
			Annotations:  annotationsJSON,
			StartsAt:     in.StartsAt,
			EndsAt:       in.EndsAt,
			GeneratorURL: in.GeneratorURL,
		}
		if err := tx.Create(&signal).Error; err != nil {
			return err
		}
		payload := map[string]any{"status": in.Status, "alert_name": in.AlertName, "route_key": in.RouteKey}
		return tx.Create(&IncidentEvent{IncidentID: incident.ID, EventType: "alert_ingested", Message: "alert signal ingested", Payload: mustJSON(payload)}).Error
	})
	if err != nil {
		return Incident{}, err
	}
	return incident, nil
}

func normalizeIncidentStatus(alertStatus string) string {
	if strings.EqualFold(alertStatus, "resolved") {
		return "resolved"
	}
	return "open"
}

func (r *Repository) ListIncidents(ctx context.Context, limit int) ([]IncidentListItem, error) {
	if limit <= 0 {
		limit = 100
	}
	var incidents []Incident
	if err := r.db.WithContext(ctx).Order("updated_at desc").Limit(limit).Find(&incidents).Error; err != nil {
		return nil, err
	}
	out := make([]IncidentListItem, 0, len(incidents))
	for _, inc := range incidents {
		total, firing, err := r.IncidentAlertStats(ctx, inc.ID)
		if err != nil {
			return nil, err
		}
		closureReady := false
		if len(inc.ClosureCriteria) > 0 {
			var snap ClosureCriteriaSnapshot
			if json.Unmarshal(inc.ClosureCriteria, &snap) == nil {
				closureReady = snap.ClosureReady
			}
		}
		out = append(out, IncidentListItem{Incident: inc, AlertsTotal: total, AlertsFiring: firing, ClosureReady: closureReady})
	}
	return out, nil
}

func (r *Repository) GetIncident(ctx context.Context, id uint) (IncidentWithSteps, error) {
	var inc Incident
	if err := r.db.WithContext(ctx).First(&inc, id).Error; err != nil {
		return IncidentWithSteps{}, err
	}
	var steps []StepRun
	if err := r.db.WithContext(ctx).Where("incident_id = ?", id).Order("created_at desc").Limit(200).Find(&steps).Error; err != nil {
		return IncidentWithSteps{}, err
	}
	execs, err := r.ListRunbookExecutions(ctx, id)
	if err != nil {
		return IncidentWithSteps{}, err
	}
	alerts, err := r.ListIncidentAlerts(ctx, id)
	if err != nil {
		return IncidentWithSteps{}, err
	}
	return IncidentWithSteps{Incident: inc, Steps: steps, Executions: execs, Alerts: alerts}, nil
}

func (r *Repository) GetOpenIncidents(ctx context.Context, limit int) ([]Incident, error) {
	if limit <= 0 {
		limit = 20
	}
	var out []Incident
	err := r.db.WithContext(ctx).
		Where("closure_state <> ?", "closed").
		Order("updated_at asc").
		Limit(limit).
		Find(&out).Error
	return out, err
}

func (r *Repository) SaveStepRuns(ctx context.Context, incidentID uint, runs []StepRunInput) error {
	if len(runs) == 0 {
		return nil
	}
	models := make([]StepRun, 0, len(runs))
	for _, run := range runs {
		models = append(models, StepRun{
			IncidentID: incidentID,
			StepName:   run.StepName,
			Tool:       run.Tool,
			Status:     run.Status,
			Output:     run.Output,
			Error:      run.Error,
			StartedAt:  run.StartedAt,
			FinishedAt: run.FinishedAt,
		})
	}
	return r.db.WithContext(ctx).Create(&models).Error
}

func (r *Repository) UpdateIncidentBrief(ctx context.Context, incidentID uint, briefText, runbookName, status string) error {
	updates := map[string]any{
		"brief":        briefText,
		"runbook_name": runbookName,
		"status":       status,
	}
	if status == "resolved" {
		now := time.Now().UTC()
		updates["closed_at"] = &now
		updates["resolved_at"] = &now
	}
	return r.db.WithContext(ctx).Model(&Incident{}).Where("id = ?", incidentID).Updates(updates).Error
}

func (r *Repository) CheckAndUpdateDedup(ctx context.Context, incidentID uint, destinationID, contentHash string, cooldown time.Duration, force bool, now time.Time) (bool, error) {
	return r.CheckAndUpdateDedupWithExecution(ctx, incidentID, destinationID, contentHash, "", cooldown, force, now)
}

func (r *Repository) CheckAndUpdateDedupWithExecution(ctx context.Context, incidentID uint, destinationID, contentHash, executionHash string, cooldown time.Duration, force bool, now time.Time) (bool, error) {
	var state DeliveryState
	shouldSend := true
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("incident_id = ? AND destination_id = ?", incidentID, destinationID).
			First(&state).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			state = DeliveryState{
				IncidentID:        incidentID,
				DestinationID:     destinationID,
				LastHash:          contentHash,
				LastExecutionHash: executionHash,
				LastSentAt:        now,
			}
			return tx.Create(&state).Error
		}
		if err != nil {
			return err
		}

		if !force && state.LastHash == contentHash && state.LastExecutionHash == executionHash && now.Sub(state.LastSentAt) < cooldown {
			shouldSend = false
			return nil
		}
		state.LastHash = contentHash
		state.LastExecutionHash = executionHash
		state.LastSentAt = now
		return tx.Save(&state).Error
	})
	if err != nil {
		return false, err
	}
	return shouldSend, nil
}

func HashContent(content string) string {
	s := sha256.Sum256([]byte(content))
	return hex.EncodeToString(s[:])
}

func (r *Repository) SetOverride(ctx context.Context, key, value string, encrypted bool) error {
	override := SettingsOverride{Key: key, Value: value, Encrypted: encrypted, UpdatedAt: time.Now().UTC()}
	return r.db.WithContext(ctx).Save(&override).Error
}

func (r *Repository) GetOverride(ctx context.Context, key string) (SettingsOverride, error) {
	var out SettingsOverride
	err := r.db.WithContext(ctx).Where("key = ?", key).First(&out).Error
	return out, err
}

func (r *Repository) DeleteOverride(ctx context.Context, key string) error {
	return r.db.WithContext(ctx).Where("key = ?", key).Delete(&SettingsOverride{}).Error
}

func (r *Repository) GetRunbooks(ctx context.Context) ([]Runbook, error) {
	var rows []Runbook
	err := r.db.WithContext(ctx).Where("enabled = ?", true).Find(&rows).Error
	return rows, err
}

func (r *Repository) GetIncidentByID(ctx context.Context, id uint) (Incident, error) {
	var out Incident
	err := r.db.WithContext(ctx).First(&out, id).Error
	return out, err
}

func (r *Repository) ListAlerts(ctx context.Context, limit int) ([]Signal, error) {
	if limit <= 0 {
		limit = 200
	}
	var out []Signal
	err := r.db.WithContext(ctx).Order("created_at desc").Limit(limit).Find(&out).Error
	return out, err
}

func (r *Repository) GetAlert(ctx context.Context, id uint) (Signal, error) {
	var out Signal
	err := r.db.WithContext(ctx).First(&out, id).Error
	return out, err
}

func (r *Repository) ListIncidentAlerts(ctx context.Context, incidentID uint) ([]Signal, error) {
	var out []Signal
	err := r.db.WithContext(ctx).Where("incident_id = ?", incidentID).Order("created_at desc").Find(&out).Error
	return out, err
}

func (r *Repository) EnsureRunbookExecution(ctx context.Context, incidentID uint, runbookName, runbookVersionHash, trigger string, plan []ExecutionStepInput) (RunbookExecution, bool, error) {
	var exec RunbookExecution
	created := false
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("incident_id = ? AND active = ?", incidentID, true).
			Order("created_at desc").
			First(&exec).Error
		if err == nil {
			return nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		now := time.Now().UTC()
		exec = RunbookExecution{
			IncidentID:         incidentID,
			RunbookName:        runbookName,
			RunbookVersionHash: runbookVersionHash,
			Status:             "running",
			StartedAt:          now,
			Trigger:            trigger,
			Active:             true,
		}
		if err := tx.Create(&exec).Error; err != nil {
			return err
		}
		if len(plan) > 0 {
			rows := make([]RunbookExecutionStep, 0, len(plan))
			for _, item := range plan {
				rows = append(rows, RunbookExecutionStep{
					ExecutionID:    exec.ID,
					StepName:       item.StepName,
					StepOrder:      item.StepOrder,
					Kind:           item.Kind,
					Status:         nonEmptyStatus(item.Status, "pending"),
					Recommendation: item.Recommendation,
					Required:       item.Required,
				})
			}
			if err := tx.Create(&rows).Error; err != nil {
				return err
			}
		}
		created = true
		return tx.Create(&IncidentEvent{IncidentID: incidentID, EventType: "runbook_started", Message: "runbook execution started", Payload: mustJSON(map[string]any{"execution_id": exec.ID, "runbook": runbookName, "trigger": trigger})}).Error
	})
	if err != nil {
		return RunbookExecution{}, false, err
	}
	return exec, created, nil
}

func nonEmptyStatus(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func (r *Repository) ListExecutionSteps(ctx context.Context, executionID uint) ([]RunbookExecutionStep, error) {
	var out []RunbookExecutionStep
	err := r.db.WithContext(ctx).Where("execution_id = ?", executionID).Order("step_order asc").Find(&out).Error
	return out, err
}

func (r *Repository) UpsertExecutionStep(ctx context.Context, executionID uint, in ExecutionStepInput) error {
	step := RunbookExecutionStep{
		ExecutionID:    executionID,
		StepName:       in.StepName,
		StepOrder:      in.StepOrder,
		Kind:           in.Kind,
		Status:         in.Status,
		Output:         in.Output,
		Error:          in.Error,
		Recommendation: in.Recommendation,
		Required:       in.Required,
		StartedAt:      in.StartedAt,
		FinishedAt:     in.FinishedAt,
	}
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "execution_id"}, {Name: "step_order"}},
		DoUpdates: clause.AssignmentColumns([]string{"step_name", "kind", "status", "output", "error", "recommendation", "required", "started_at", "finished_at", "updated_at"}),
	}).Create(&step).Error
}

func (r *Repository) MarkExecutionFinished(ctx context.Context, executionID uint, status string) error {
	now := time.Now().UTC()
	return r.db.WithContext(ctx).Model(&RunbookExecution{}).Where("id = ?", executionID).Updates(map[string]any{
		"status":      status,
		"active":      false,
		"finished_at": &now,
	}).Error
}

func (r *Repository) UpdateExecutionStatus(ctx context.Context, executionID uint, status string, active bool) error {
	updates := map[string]any{"status": status, "active": active}
	if !active && (status == "completed" || status == "failed" || status == "blocked") {
		now := time.Now().UTC()
		updates["finished_at"] = &now
	}
	return r.db.WithContext(ctx).Model(&RunbookExecution{}).Where("id = ?", executionID).Updates(updates).Error
}

func (r *Repository) ListRunbookExecutions(ctx context.Context, incidentID uint) ([]RunbookExecutionWithSteps, error) {
	var execs []RunbookExecution
	if err := r.db.WithContext(ctx).Where("incident_id = ?", incidentID).Order("created_at desc").Find(&execs).Error; err != nil {
		return nil, err
	}
	out := make([]RunbookExecutionWithSteps, 0, len(execs))
	for _, ex := range execs {
		steps, err := r.ListExecutionSteps(ctx, ex.ID)
		if err != nil {
			return nil, err
		}
		out = append(out, RunbookExecutionWithSteps{Execution: ex, Steps: steps})
	}
	return out, nil
}

func (r *Repository) GetLatestRunbookExecution(ctx context.Context, incidentID uint) (RunbookExecutionWithSteps, error) {
	var ex RunbookExecution
	if err := r.db.WithContext(ctx).Where("incident_id = ?", incidentID).Order("created_at desc").First(&ex).Error; err != nil {
		return RunbookExecutionWithSteps{}, err
	}
	steps, err := r.ListExecutionSteps(ctx, ex.ID)
	if err != nil {
		return RunbookExecutionWithSteps{}, err
	}
	return RunbookExecutionWithSteps{Execution: ex, Steps: steps}, nil
}

func (r *Repository) ReRunIncidentRunbook(ctx context.Context, incidentID uint) error {
	return r.db.WithContext(ctx).Model(&RunbookExecution{}).Where("incident_id = ? AND active = ?", incidentID, true).Updates(map[string]any{
		"active":      false,
		"status":      "superseded",
		"finished_at": time.Now().UTC(),
	}).Error
}

func (r *Repository) RecordIncidentEvent(ctx context.Context, incidentID uint, eventType, message string, payload any) error {
	blob := mustJSON(payload)
	return r.db.WithContext(ctx).Create(&IncidentEvent{
		IncidentID: incidentID,
		EventType:  eventType,
		Message:    message,
		Payload:    blob,
	}).Error
}

func (r *Repository) ListIncidentEvents(ctx context.Context, incidentID uint, limit int) ([]IncidentEvent, error) {
	if limit <= 0 {
		limit = 200
	}
	var out []IncidentEvent
	err := r.db.WithContext(ctx).Where("incident_id = ?", incidentID).Order("created_at desc").Limit(limit).Find(&out).Error
	return out, err
}

func (r *Repository) IncidentAlertStats(ctx context.Context, incidentID uint) (int, int, error) {
	alerts, err := r.ListIncidentAlerts(ctx, incidentID)
	if err != nil {
		return 0, 0, err
	}
	state := latestAlertStates(alerts)
	total := len(state)
	firing := 0
	for _, status := range state {
		if strings.EqualFold(status, "firing") || strings.EqualFold(status, "open") {
			firing++
		}
	}
	return total, firing, nil
}

func latestAlertStates(alerts []Signal) map[string]string {
	sort.SliceStable(alerts, func(i, j int) bool {
		return alerts[i].CreatedAt.After(alerts[j].CreatedAt)
	})
	latest := map[string]string{}
	for _, sig := range alerts {
		key := signalAggregationKey(sig)
		if _, exists := latest[key]; exists {
			continue
		}
		latest[key] = strings.ToLower(strings.TrimSpace(sig.Status))
	}
	return latest
}

func signalAggregationKey(sig Signal) string {
	labels := map[string]string{}
	_ = json.Unmarshal(sig.Labels, &labels)
	alert := nonEmpty(labels["alertname"], sig.AlertName)
	instance := nonEmpty(labels["instance"], labels["job"])
	if instance == "" {
		instance = nonEmpty(labels["pod"], labels["node"])
	}
	if instance == "" {
		instance = "global"
	}
	return fmt.Sprintf("%s|%s", alert, instance)
}

func nonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func (r *Repository) UpdateIncidentClosure(ctx context.Context, incidentID uint, snap ClosureCriteriaSnapshot, status, closureState, closureReason string, streak int) error {
	blob, err := json.Marshal(snap)
	if err != nil {
		return err
	}
	updates := map[string]any{
		"closure_criteria":     blob,
		"closure_state":        closureState,
		"closure_reason":       closureReason,
		"closure_ready_streak": streak,
		"status":               status,
	}
	if closureState == "closed" {
		now := time.Now().UTC()
		updates["closed_at"] = &now
		updates["resolved_at"] = &now
	}
	if closureState == "open" || closureState == "reopened" || closureState == "ready_to-close" {
		updates["closed_at"] = nil
	}
	if err := r.db.WithContext(ctx).Model(&Incident{}).Where("id = ?", incidentID).Updates(updates).Error; err != nil {
		return err
	}
	return r.RecordIncidentEvent(ctx, incidentID, "criteria_changed", "closure criteria updated", snap)
}

func (r *Repository) ManualCloseIncident(ctx context.Context, incidentID uint, reason string) error {
	now := time.Now().UTC()
	updates := map[string]any{
		"status":               "resolved",
		"closure_state":        "closed",
		"closure_reason":       "manual:" + strings.TrimSpace(reason),
		"closed_at":            &now,
		"resolved_at":          &now,
		"closure_ready_streak": 0,
	}
	if err := r.db.WithContext(ctx).Model(&Incident{}).Where("id = ?", incidentID).Updates(updates).Error; err != nil {
		return err
	}
	return r.RecordIncidentEvent(ctx, incidentID, "manual_close", "incident closed manually", map[string]any{"reason": reason})
}

func (r *Repository) ManualReopenIncident(ctx context.Context, incidentID uint, reason string) error {
	updates := map[string]any{
		"status":               "open",
		"closure_state":        "reopened",
		"closure_reason":       "manual_reopen:" + strings.TrimSpace(reason),
		"closed_at":            nil,
		"resolved_at":          nil,
		"closure_ready_streak": 0,
	}
	if err := r.db.WithContext(ctx).Model(&Incident{}).Where("id = ?", incidentID).Updates(updates).Error; err != nil {
		return err
	}
	return r.RecordIncidentEvent(ctx, incidentID, "manual_reopen", "incident reopened manually", map[string]any{"reason": reason})
}

func (r *Repository) GetIncidentClosureCriteria(ctx context.Context, incidentID uint) (ClosureCriteriaSnapshot, error) {
	inc, err := r.GetIncidentByID(ctx, incidentID)
	if err != nil {
		return ClosureCriteriaSnapshot{}, err
	}
	if len(inc.ClosureCriteria) == 0 {
		return ClosureCriteriaSnapshot{}, nil
	}
	var out ClosureCriteriaSnapshot
	if err := json.Unmarshal(inc.ClosureCriteria, &out); err != nil {
		return ClosureCriteriaSnapshot{}, err
	}
	return out, nil
}

func (r *Repository) GetOrCreateApproval(ctx context.Context, incidentID, executionID uint, stepOrder int, action, reason string) (ApprovalRequest, error) {
	var req ApprovalRequest
	err := r.db.WithContext(ctx).Where(
		"incident_id = ? AND execution_id = ? AND step_order = ? AND action = ? AND status = ?",
		incidentID, executionID, stepOrder, action, "pending",
	).Order("created_at desc").First(&req).Error
	if err == nil {
		return req, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return ApprovalRequest{}, err
	}
	req = ApprovalRequest{
		IncidentID:  incidentID,
		ExecutionID: executionID,
		StepOrder:   stepOrder,
		Action:      action,
		Status:      "pending",
		Reason:      reason,
	}
	if err := r.db.WithContext(ctx).Create(&req).Error; err != nil {
		return ApprovalRequest{}, err
	}
	if err := r.RecordIncidentEvent(ctx, incidentID, "approval_created", "approval request created", map[string]any{"approval_id": req.ID, "execution_id": executionID, "step_order": stepOrder}); err != nil {
		return ApprovalRequest{}, err
	}
	return req, nil
}

func (r *Repository) UpdateApprovalStatus(ctx context.Context, id uint, status, reason string) error {
	if err := r.db.WithContext(ctx).Model(&ApprovalRequest{}).Where("id = ?", id).Updates(map[string]any{"status": status, "reason": reason}).Error; err != nil {
		return err
	}
	var req ApprovalRequest
	if err := r.db.WithContext(ctx).First(&req, id).Error; err != nil {
		return err
	}
	return r.RecordIncidentEvent(ctx, req.IncidentID, "approval_"+status, "approval status updated", map[string]any{"approval_id": id, "status": status, "reason": reason})
}

func (r *Repository) GetApproval(ctx context.Context, id uint) (ApprovalRequest, error) {
	var req ApprovalRequest
	err := r.db.WithContext(ctx).First(&req, id).Error
	return req, err
}

func (r *Repository) ListPendingApprovals(ctx context.Context, limit int) ([]ApprovalRequest, error) {
	if limit <= 0 {
		limit = 100
	}
	var out []ApprovalRequest
	err := r.db.WithContext(ctx).Where("status = ?", "pending").Order("created_at asc").Limit(limit).Find(&out).Error
	return out, err
}

func (r *Repository) CreateGitOpsChange(ctx context.Context, in GitOpsChangeInput) (GitOpsChange, error) {
	desired := mustJSON(in.Desired)
	row := GitOpsChange{
		ChangeType:  in.ChangeType,
		Title:       in.Title,
		Status:      "pr_open",
		Branch:      fmt.Sprintf("rh-change-%d", time.Now().UTC().Unix()),
		PRURL:       "",
		PRNumber:    0,
		Desired:     desired,
		Applied:     []byte("{}"),
		DriftStatus: "pending",
	}
	if err := r.db.WithContext(ctx).Create(&row).Error; err != nil {
		return GitOpsChange{}, err
	}
	return row, nil
}

func (r *Repository) ListGitOpsChanges(ctx context.Context, limit int) ([]GitOpsChange, error) {
	if limit <= 0 {
		limit = 100
	}
	var out []GitOpsChange
	err := r.db.WithContext(ctx).Order("created_at desc").Limit(limit).Find(&out).Error
	return out, err
}

func (r *Repository) GetGitOpsChange(ctx context.Context, id uint) (GitOpsChange, error) {
	var out GitOpsChange
	err := r.db.WithContext(ctx).First(&out, id).Error
	return out, err
}

func mustJSON(payload any) []byte {
	blob, err := json.Marshal(payload)
	if err != nil {
		return []byte("{}")
	}
	return blob
}
