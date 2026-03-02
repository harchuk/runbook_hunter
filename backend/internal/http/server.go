package http

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"gorm.io/gorm"

	"github.com/runbook-hunter/runbook-hunter/backend/internal/app"
	"github.com/runbook-hunter/runbook-hunter/backend/internal/authn"
	"github.com/runbook-hunter/runbook-hunter/backend/internal/correlation"
	"github.com/runbook-hunter/runbook-hunter/backend/internal/ingest"
	"github.com/runbook-hunter/runbook-hunter/backend/internal/obs"
	"github.com/runbook-hunter/runbook-hunter/backend/internal/store"
)

type Server struct {
	app        *app.App
	httpServer *http.Server
}

func NewServer(a *app.App) *Server {
	r := chi.NewRouter()
	r.Use(chimw.RequestID)
	r.Use(chimw.RealIP)
	r.Use(chimw.Recoverer)
	r.Use(chimw.Timeout(60 * time.Second))
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:3000", "https://localhost:3000"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	r.Get("/healthz", obs.Healthz())
	r.Get("/readyz", obs.Readyz(a.Repo.Ping))
	r.Handle("/metrics", promhttp.Handler())

	h := &handler{app: a}
	r.Post("/api/alertmanager", h.postAlertmanager)

	r.Route("/api", func(api chi.Router) {
		api.Use(authMiddleware(a))

		api.Get("/alerts", h.getAlerts)
		api.Get("/alerts/{id}", h.getAlert)

		api.Get("/incidents", h.getIncidents)
		api.Get("/incidents/{id}", h.getIncident)
		api.Get("/incidents/{id}/alerts", h.getIncidentAlerts)
		api.Get("/incidents/{id}/runbook-executions", h.getIncidentRunbookExecutions)
		api.Get("/incidents/{id}/closure-criteria", h.getIncidentClosureCriteria)
		api.Get("/incidents/{id}/events", h.getIncidentEvents)
		api.Post("/incidents/{id}/post-update", h.postUpdateNow)
		api.Post("/incidents/{id}/close", h.postCloseIncident)
		api.Post("/incidents/{id}/reopen", h.postReopenIncident)
		api.Post("/incidents/{id}/runbook/rerun", h.postRerunRunbook)

		api.Get("/settings/effective", h.getSettingsEffective)
		api.Get("/settings/overrides", h.getSettingsOverrides)
		api.Put("/settings/overrides", h.putSettingsOverrides)
		api.Post("/settings/overrides/reset", h.resetSettingsOverrides)

		api.Get("/approvals", h.getApprovals)
		api.Post("/approvals/{id}/approve", h.approveRequest)
		api.Post("/approvals/{id}/reject", h.rejectRequest)

		api.Post("/gitops/changes", h.createGitOpsChange)
		api.Get("/gitops/changes", h.listGitOpsChanges)
		api.Get("/gitops/changes/{id}", h.getGitOpsChange)
	})

	server := &http.Server{
		Addr:         a.Config.Server.Addr,
		Handler:      r,
		ReadTimeout:  a.Config.Server.ReadTimeout,
		WriteTimeout: a.Config.Server.WriteTimeout,
	}
	return &Server{app: a, httpServer: server}
}

func (s *Server) Start() error {
	s.app.Logger.Info().Str("addr", s.httpServer.Addr).Msg("api server started")
	return s.httpServer.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}

type handler struct {
	app *app.App
}

func (h *handler) postAlertmanager(w http.ResponseWriter, r *http.Request) {
	payload, err := ingest.ParseAlertmanagerPayload(r.Body)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	cfg, err := h.app.Settings.EffectiveConfig(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	processed := 0
	for _, alert := range payload.Alerts {
		labels := alert.Labels
		if labels == nil {
			labels = map[string]string{}
		}
		routeKey := h.app.Router.ComputeRouteKey(labels, cfg)
		fingerprint, _ := correlation.Build(labels, routeKey, cfg.Fingerprint.Fields)
		_, err := h.app.Repo.UpsertIncidentAndSignal(r.Context(), store.AlertSignalInput{
			Fingerprint:  fingerprint,
			RouteKey:     routeKey,
			AlertName:    labels["alertname"],
			Status:       alert.Status,
			Labels:       labels,
			Annotations:  alert.Annotations,
			StartsAt:     alert.StartsAt,
			EndsAt:       alert.EndsAt,
			GeneratorURL: alert.GeneratorURL,
		})
		if err != nil {
			h.app.Metrics.ErrorsTotal.WithLabelValues("ingest").Inc()
			h.app.Logger.Error().Err(err).Msg("ingest failed for alert")
			continue
		}
		h.app.Metrics.SignalsIngested.Inc()
		processed++
	}

	writeJSON(w, http.StatusAccepted, map[string]any{"processed": processed, "total": len(payload.Alerts)})
}

func (h *handler) getAlerts(w http.ResponseWriter, r *http.Request) {
	principal := principalFromRequest(r)
	alerts, err := h.app.Repo.ListAlerts(r.Context(), 500)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	filtered := make([]store.Signal, 0, len(alerts))
	for _, alert := range alerts {
		if !h.app.Access.CanViewSignal(principal, alert) {
			continue
		}
		filtered = append(filtered, alert)
	}
	writeJSON(w, http.StatusOK, mapSignalsResponse(filtered))
}

func (h *handler) getAlert(w http.ResponseWriter, r *http.Request) {
	principal := principalFromRequest(r)
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	alert, err := h.app.Repo.GetAlert(r.Context(), id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if !h.app.Access.CanViewSignal(principal, alert) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return
	}
	writeJSON(w, http.StatusOK, mapSignalResponse(alert))
}

func (h *handler) getIncidents(w http.ResponseWriter, r *http.Request) {
	principal := principalFromRequest(r)
	incidents, err := h.app.Repo.ListIncidents(r.Context(), 200)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	response := make([]map[string]any, 0, len(incidents))
	for _, item := range incidents {
		if !h.app.Access.CanViewIncident(principal, item.Incident) {
			continue
		}
		response = append(response, map[string]any{
			"id":            item.ID,
			"createdAt":     item.CreatedAt,
			"updatedAt":     item.UpdatedAt,
			"alertName":     item.AlertName,
			"service":       item.Service,
			"env":           item.Env,
			"severity":      item.Severity,
			"status":        item.Status,
			"runbookName":   item.RunbookName,
			"brief":         item.Brief,
			"closureState":  item.ClosureState,
			"closureReason": item.ClosureReason,
			"alerts_total":  item.AlertsTotal,
			"alerts_firing": item.AlertsFiring,
			"closure_ready": item.ClosureReady,
		})
	}
	writeJSON(w, http.StatusOK, response)
}

func (h *handler) getIncident(w http.ResponseWriter, r *http.Request) {
	principal := principalFromRequest(r)
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	incident, err := h.app.Repo.GetIncident(r.Context(), id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if !h.app.Access.CanViewIncident(principal, incident.Incident) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return
	}
	response := map[string]any{
		"incident":   mapIncidentResponse(incident.Incident),
		"steps":      incident.Steps,
		"executions": incident.Executions,
		"alerts":     mapSignalsResponse(incident.Alerts),
	}
	writeJSON(w, http.StatusOK, response)
}

func (h *handler) getIncidentAlerts(w http.ResponseWriter, r *http.Request) {
	principal := principalFromRequest(r)
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	incident, err := h.app.Repo.GetIncidentByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if !h.app.Access.CanViewIncident(principal, incident) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return
	}
	alerts, err := h.app.Repo.ListIncidentAlerts(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	filtered := make([]store.Signal, 0, len(alerts))
	for _, alert := range alerts {
		if !h.app.Access.CanViewSignal(principal, alert) {
			continue
		}
		filtered = append(filtered, alert)
	}
	writeJSON(w, http.StatusOK, mapSignalsResponse(filtered))
}

func (h *handler) getIncidentRunbookExecutions(w http.ResponseWriter, r *http.Request) {
	principal := principalFromRequest(r)
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	incident, err := h.app.Repo.GetIncidentByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if !h.app.Access.CanViewIncident(principal, incident) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return
	}
	execs, err := h.app.Repo.ListRunbookExecutions(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, execs)
}

func (h *handler) getIncidentClosureCriteria(w http.ResponseWriter, r *http.Request) {
	principal := principalFromRequest(r)
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	incident, err := h.app.Repo.GetIncidentByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if !h.app.Access.CanViewIncident(principal, incident) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return
	}
	criteria, err := h.app.Repo.GetIncidentClosureCriteria(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, criteria)
}

func (h *handler) getIncidentEvents(w http.ResponseWriter, r *http.Request) {
	principal := principalFromRequest(r)
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	incident, err := h.app.Repo.GetIncidentByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if !h.app.Access.CanViewIncident(principal, incident) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return
	}
	events, err := h.app.Repo.ListIncidentEvents(r.Context(), id, 300)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, events)
}

func (h *handler) postUpdateNow(w http.ResponseWriter, r *http.Request) {
	principal := principalFromRequest(r)
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	incident, err := h.app.Repo.GetIncidentByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if !h.app.Access.CanViewIncident(principal, incident) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return
	}
	if !h.app.Access.CanWrite(principal) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "admin role required"})
		return
	}
	var body struct {
		DestinationIDs []string `json:"destinationIds"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)

	if err := h.app.Worker.PostUpdateNow(r.Context(), id, body.DestinationIDs); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "queued"})
}

func (h *handler) postCloseIncident(w http.ResponseWriter, r *http.Request) {
	principal := principalFromRequest(r)
	if !h.app.Access.CanWrite(principal) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "admin role required"})
		return
	}
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	var body struct {
		Reason string `json:"reason"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	if strings.TrimSpace(body.Reason) == "" {
		body.Reason = "operator close"
	}
	if err := h.app.Repo.ManualCloseIncident(r.Context(), id, body.Reason); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "closed"})
}

func (h *handler) postReopenIncident(w http.ResponseWriter, r *http.Request) {
	principal := principalFromRequest(r)
	if !h.app.Access.CanWrite(principal) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "admin role required"})
		return
	}
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	var body struct {
		Reason string `json:"reason"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	if strings.TrimSpace(body.Reason) == "" {
		body.Reason = "operator reopen"
	}
	if err := h.app.Repo.ManualReopenIncident(r.Context(), id, body.Reason); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "reopened"})
}

func (h *handler) postRerunRunbook(w http.ResponseWriter, r *http.Request) {
	principal := principalFromRequest(r)
	if !h.app.Access.CanWrite(principal) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "admin role required"})
		return
	}
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	if err := h.app.Repo.ReRunIncidentRunbook(r.Context(), id); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if err := h.app.Worker.PostUpdateNow(r.Context(), id, nil); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "rerun_started"})
}

func (h *handler) getSettingsEffective(w http.ResponseWriter, r *http.Request) {
	principal := principalFromRequest(r)
	if !h.app.Access.CanWrite(principal) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "admin role required"})
		return
	}
	cfg, err := h.app.Settings.EffectiveConfig(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, cfg)
}

func (h *handler) getSettingsOverrides(w http.ResponseWriter, r *http.Request) {
	principal := principalFromRequest(r)
	if !h.app.Access.CanWrite(principal) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "admin role required"})
		return
	}
	overrides, err := h.app.Settings.GetOverrides(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, overrides)
}

func (h *handler) putSettingsOverrides(w http.ResponseWriter, r *http.Request) {
	principal := principalFromRequest(r)
	if !h.app.Access.CanWrite(principal) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "admin role required"})
		return
	}
	strict, err := h.app.Settings.IsGitOpsStrict(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if strict {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "gitops strict mode enabled", "hint": "use POST /api/gitops/changes"})
		return
	}

	var payload map[string]any
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	if err := h.app.Settings.PutOverrides(r.Context(), payload); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "saved"})
}

func (h *handler) resetSettingsOverrides(w http.ResponseWriter, r *http.Request) {
	principal := principalFromRequest(r)
	if !h.app.Access.CanWrite(principal) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "admin role required"})
		return
	}
	strict, err := h.app.Settings.IsGitOpsStrict(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if strict {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "gitops strict mode enabled", "hint": "use POST /api/gitops/changes"})
		return
	}

	var payload struct {
		Scope string   `json:"scope"`
		Keys  []string `json:"keys"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	switch payload.Scope {
	case "all":
		if err := h.app.Settings.ResetAll(r.Context()); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
	case "keys":
		if err := h.app.Settings.ResetKeys(r.Context(), payload.Keys); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
	default:
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "scope must be all or keys"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "reset"})
}

func (h *handler) getApprovals(w http.ResponseWriter, r *http.Request) {
	principal := principalFromRequest(r)
	if !h.app.Access.CanWrite(principal) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "admin role required"})
		return
	}
	rows, err := h.app.Repo.ListPendingApprovals(r.Context(), 200)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, rows)
}

func (h *handler) approveRequest(w http.ResponseWriter, r *http.Request) {
	principal := principalFromRequest(r)
	if !h.app.Access.CanWrite(principal) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "admin role required"})
		return
	}
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	if err := h.app.Repo.UpdateApprovalStatus(r.Context(), id, "approved", "approved from api"); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "approved"})
}

func (h *handler) rejectRequest(w http.ResponseWriter, r *http.Request) {
	principal := principalFromRequest(r)
	if !h.app.Access.CanWrite(principal) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "admin role required"})
		return
	}
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	var body struct {
		Reason string `json:"reason"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	if strings.TrimSpace(body.Reason) == "" {
		body.Reason = "rejected"
	}
	if err := h.app.Repo.UpdateApprovalStatus(r.Context(), id, "rejected", body.Reason); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "rejected"})
}

func (h *handler) createGitOpsChange(w http.ResponseWriter, r *http.Request) {
	principal := principalFromRequest(r)
	if !h.app.Access.CanWrite(principal) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "admin role required"})
		return
	}
	var payload struct {
		ChangeType string         `json:"change_type"`
		Title      string         `json:"title"`
		Desired    map[string]any `json:"desired"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	if strings.TrimSpace(payload.ChangeType) == "" {
		payload.ChangeType = "settings"
	}
	if strings.TrimSpace(payload.Title) == "" {
		payload.Title = "Runbook Hunter change request"
	}
	row, err := h.app.Repo.CreateGitOpsChange(r.Context(), store.GitOpsChangeInput{
		ChangeType: payload.ChangeType,
		Title:      payload.Title,
		Desired:    payload.Desired,
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, row)
}

func (h *handler) listGitOpsChanges(w http.ResponseWriter, r *http.Request) {
	principal := principalFromRequest(r)
	if !h.app.Access.CanWrite(principal) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "admin role required"})
		return
	}
	rows, err := h.app.Repo.ListGitOpsChanges(r.Context(), 200)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, rows)
}

func (h *handler) getGitOpsChange(w http.ResponseWriter, r *http.Request) {
	principal := principalFromRequest(r)
	if !h.app.Access.CanWrite(principal) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "admin role required"})
		return
	}
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	row, err := h.app.Repo.GetGitOpsChange(r.Context(), id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, row)
}

func authMiddleware(a *app.App) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			principal, err := a.AuthN.AuthenticateRequest(r)
			if err != nil {
				if strings.ToLower(strings.TrimSpace(a.Config.Auth.Mode)) == "basic" {
					w.Header().Set("WWW-Authenticate", `Basic realm="runbook-hunter"`)
				}
				writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
				return
			}
			next.ServeHTTP(w, r.WithContext(authn.WithPrincipal(r.Context(), principal)))
		})
	}
}

func parseID(value string) (uint, error) {
	parsed, err := strconv.ParseUint(value, 10, 32)
	if err != nil {
		return 0, err
	}
	return uint(parsed), nil
}

func writeJSON(w http.ResponseWriter, code int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(payload)
}

func mapSignalsResponse(items []store.Signal) []map[string]any {
	out := make([]map[string]any, 0, len(items))
	for _, item := range items {
		out = append(out, mapSignalResponse(item))
	}
	return out
}

func mapSignalResponse(item store.Signal) map[string]any {
	return map[string]any{
		"id":           item.ID,
		"createdAt":    item.CreatedAt,
		"incidentId":   item.IncidentID,
		"fingerprint":  item.Fingerprint,
		"alertName":    item.AlertName,
		"status":       item.Status,
		"labels":       decodeJSONMap(item.Labels),
		"annotations":  decodeJSONMap(item.Annotations),
		"startsAt":     item.StartsAt,
		"endsAt":       item.EndsAt,
		"generatorUrl": item.GeneratorURL,
	}
}

func mapIncidentResponse(item store.Incident) map[string]any {
	return map[string]any{
		"id":                 item.ID,
		"createdAt":          item.CreatedAt,
		"updatedAt":          item.UpdatedAt,
		"fingerprint":        item.Fingerprint,
		"routeKey":           item.RouteKey,
		"alertName":          item.AlertName,
		"service":            item.Service,
		"env":                item.Env,
		"severity":           item.Severity,
		"status":             item.Status,
		"runbookName":        item.RunbookName,
		"brief":              item.Brief,
		"openedAt":           item.OpenedAt,
		"closedAt":           item.ClosedAt,
		"lastSignalAt":       item.LastSignalAt,
		"labels":             decodeJSONMap(item.Labels),
		"closureState":       item.ClosureState,
		"closureReason":      item.ClosureReason,
		"closureCriteria":    decodeJSONAny(item.ClosureCriteria),
		"resolvedAt":         item.ResolvedAt,
		"closureReadyStreak": item.ClosureReadyStreak,
	}
}

func decodeJSONMap(raw []byte) map[string]any {
	if len(raw) == 0 {
		return map[string]any{}
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil || out == nil {
		return map[string]any{}
	}
	return out
}

func decodeJSONAny(raw []byte) any {
	if len(raw) == 0 {
		return map[string]any{}
	}
	var out any
	if err := json.Unmarshal(raw, &out); err != nil {
		return map[string]any{}
	}
	return out
}

func principalFromRequest(r *http.Request) authn.Principal {
	principal, ok := authn.PrincipalFromContext(r.Context())
	if !ok {
		return authn.Principal{Subject: "anonymous", Username: "anonymous", IsAdmin: false}
	}
	return principal
}
