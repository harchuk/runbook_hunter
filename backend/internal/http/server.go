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
	"github.com/golang-jwt/jwt/v5"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"gorm.io/gorm"

	"github.com/runbook-hunter/runbook-hunter/backend/internal/app"
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

	r.Post("/api/alertmanager", func(w http.ResponseWriter, req *http.Request) {
		h := &handler{app: a}
		h.postAlertmanager(w, req)
	})

	r.Route("/api", func(api chi.Router) {
		api.Use(authMiddleware(a))
		h := &handler{app: a}
		api.Get("/incidents", h.getIncidents)
		api.Get("/incidents/{id}", h.getIncident)
		api.Post("/incidents/{id}/post-update", h.postUpdateNow)
		api.Get("/settings/effective", h.getSettingsEffective)
		api.Get("/settings/overrides", h.getSettingsOverrides)
		api.Put("/settings/overrides", h.putSettingsOverrides)
		api.Post("/settings/overrides/reset", h.resetSettingsOverrides)
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

func (h *handler) getIncidents(w http.ResponseWriter, r *http.Request) {
	incidents, err := h.app.Repo.ListIncidents(r.Context(), 100)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, incidents)
}

func (h *handler) getIncident(w http.ResponseWriter, r *http.Request) {
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
	writeJSON(w, http.StatusOK, incident)
}

func (h *handler) postUpdateNow(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
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

func (h *handler) getSettingsEffective(w http.ResponseWriter, r *http.Request) {
	cfg, err := h.app.Settings.EffectiveConfig(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, cfg)
}

func (h *handler) getSettingsOverrides(w http.ResponseWriter, r *http.Request) {
	overrides, err := h.app.Settings.GetOverrides(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, overrides)
}

func (h *handler) putSettingsOverrides(w http.ResponseWriter, r *http.Request) {
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

func authMiddleware(a *app.App) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			mode := strings.ToLower(a.Config.Auth.Mode)
			switch mode {
			case "jwt":
				authorization := strings.TrimSpace(r.Header.Get("Authorization"))
				if !strings.HasPrefix(authorization, "Bearer ") {
					writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "missing bearer token"})
					return
				}
				tokenString := strings.TrimPrefix(authorization, "Bearer ")
				_, err := jwt.Parse(tokenString, func(token *jwt.Token) (any, error) {
					if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
						return nil, errors.New("unexpected signing method")
					}
					return []byte(a.Config.Auth.JWTSecret), nil
				})
				if err != nil {
					writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid token"})
					return
				}
			default:
				user, pass, ok := r.BasicAuth()
				if !ok || user != a.Config.Auth.BasicUser || pass != a.Config.Auth.BasicPass {
					w.Header().Set("WWW-Authenticate", `Basic realm="runbook-hunter"`)
					writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
					return
				}
			}
			next.ServeHTTP(w, r)
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
