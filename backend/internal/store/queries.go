package store

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
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

type IncidentWithSteps struct {
	Incident Incident
	Steps    []StepRun
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
	err = r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		err := tx.Where("fingerprint = ?", in.Fingerprint).First(&incident).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			incident = Incident{
				Fingerprint:  in.Fingerprint,
				RouteKey:     in.RouteKey,
				AlertName:    in.AlertName,
				Service:      in.Labels["service"],
				Env:          in.Labels["env"],
				Severity:     in.Labels["severity"],
				Status:       normalizeIncidentStatus(in.Status),
				OpenedAt:     in.StartsAt,
				LastSignalAt: time.Now().UTC(),
				Labels:       labelsJSON,
			}
			if in.Status == "resolved" {
				now := time.Now().UTC()
				incident.ClosedAt = &now
			}
			if err := tx.Create(&incident).Error; err != nil {
				return err
			}
		} else if err != nil {
			return err
		} else {
			update := map[string]any{
				"route_key":      in.RouteKey,
				"severity":       in.Labels["severity"],
				"status":         normalizeIncidentStatus(in.Status),
				"last_signal_at": time.Now().UTC(),
				"labels":         labelsJSON,
			}
			if in.Status == "resolved" {
				now := time.Now().UTC()
				update["closed_at"] = &now
			} else {
				update["closed_at"] = nil
			}
			if err := tx.Model(&incident).Updates(update).Error; err != nil {
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
		return tx.Create(&signal).Error
	})
	if err != nil {
		return Incident{}, err
	}
	return incident, nil
}

func normalizeIncidentStatus(alertStatus string) string {
	if alertStatus == "resolved" {
		return "resolved"
	}
	return "open"
}

func (r *Repository) ListIncidents(ctx context.Context, limit int) ([]Incident, error) {
	if limit <= 0 {
		limit = 100
	}
	var out []Incident
	err := r.db.WithContext(ctx).Order("updated_at desc").Limit(limit).Find(&out).Error
	return out, err
}

func (r *Repository) GetIncident(ctx context.Context, id uint) (IncidentWithSteps, error) {
	var inc Incident
	if err := r.db.WithContext(ctx).First(&inc, id).Error; err != nil {
		return IncidentWithSteps{}, err
	}
	var steps []StepRun
	if err := r.db.WithContext(ctx).Where("incident_id = ?", id).Order("created_at desc").Limit(100).Find(&steps).Error; err != nil {
		return IncidentWithSteps{}, err
	}
	return IncidentWithSteps{Incident: inc, Steps: steps}, nil
}

func (r *Repository) GetOpenIncidents(ctx context.Context, limit int) ([]Incident, error) {
	if limit <= 0 {
		limit = 20
	}
	var out []Incident
	err := r.db.WithContext(ctx).
		Where("status = ?", "open").
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

func (r *Repository) UpdateIncidentBrief(ctx context.Context, incidentID uint, brief, runbookName, status string) error {
	updates := map[string]any{
		"brief":        brief,
		"runbook_name": runbookName,
		"status":       status,
	}
	if status == "resolved" {
		now := time.Now().UTC()
		updates["closed_at"] = &now
	}
	return r.db.WithContext(ctx).Model(&Incident{}).Where("id = ?", incidentID).Updates(updates).Error
}

func (r *Repository) CheckAndUpdateDedup(ctx context.Context, incidentID uint, destinationID, contentHash string, cooldown time.Duration, force bool, now time.Time) (bool, error) {
	var state DeliveryState
	shouldSend := true
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("incident_id = ? AND destination_id = ?", incidentID, destinationID).
			First(&state).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			state = DeliveryState{IncidentID: incidentID, DestinationID: destinationID, LastHash: contentHash, LastSentAt: now}
			return tx.Create(&state).Error
		}
		if err != nil {
			return err
		}

		if !force && state.LastHash == contentHash && now.Sub(state.LastSentAt) < cooldown {
			shouldSend = false
			return nil
		}
		state.LastHash = contentHash
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

func (r *Repository) Close() error {
	sqlDB, err := r.db.DB()
	if err != nil {
		return err
	}
	if sqlDB == nil {
		return fmt.Errorf("sql db unavailable")
	}
	return sqlDB.Close()
}
