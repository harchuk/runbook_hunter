package authz

import (
	"encoding/json"
	"slices"
	"strings"

	"github.com/runbook-hunter/runbook-hunter/backend/internal/authn"
	"github.com/runbook-hunter/runbook-hunter/backend/internal/config"
	"github.com/runbook-hunter/runbook-hunter/backend/internal/store"
)

type Engine struct {
	rules []normalizedRule
}

type normalizedRule struct {
	group      string
	services   []string
	envs       []string
	severities []string
	alertNames []string
}

func NewEngine(cfg config.AccessConfig) *Engine {
	rules := make([]normalizedRule, 0, len(cfg.GroupRules))
	for _, rule := range cfg.GroupRules {
		group := normalize(rule.Group)
		if group == "" {
			continue
		}
		rules = append(rules, normalizedRule{
			group:      group,
			services:   normalizeSlice(rule.Services),
			envs:       normalizeSlice(rule.Envs),
			severities: normalizeSlice(rule.Severities),
			alertNames: normalizeSlice(rule.AlertNames),
		})
	}
	return &Engine{rules: rules}
}

func (e *Engine) CanViewIncident(principal authn.Principal, incident store.Incident) bool {
	if principal.IsAdmin {
		return true
	}
	if len(e.rules) == 0 {
		return true
	}
	return e.match(principal, incident.Service, incident.Env, incident.Severity, incident.AlertName)
}

func (e *Engine) CanViewSignal(principal authn.Principal, signal store.Signal) bool {
	if principal.IsAdmin {
		return true
	}
	if len(e.rules) == 0 {
		return true
	}
	labels := map[string]string{}
	_ = json.Unmarshal(signal.Labels, &labels)
	return e.match(
		principal,
		nonEmpty(labels["service"]),
		nonEmpty(labels["env"]),
		nonEmpty(labels["severity"]),
		nonEmpty(labels["alertname"], signal.AlertName),
	)
}

func (e *Engine) CanWrite(principal authn.Principal) bool {
	return principal.IsAdmin
}

func (e *Engine) match(principal authn.Principal, service, env, severity, alertName string) bool {
	groups := normalizeSlice(principal.Groups)
	service = normalize(service)
	env = normalize(env)
	severity = normalize(severity)
	alertName = normalize(alertName)

	for _, rule := range e.rules {
		if !slices.Contains(groups, rule.group) {
			continue
		}
		if !matchesDimension(rule.services, service) {
			continue
		}
		if !matchesDimension(rule.envs, env) {
			continue
		}
		if !matchesDimension(rule.severities, severity) {
			continue
		}
		if !matchesDimension(rule.alertNames, alertName) {
			continue
		}
		return true
	}
	return false
}

func matchesDimension(allowed []string, value string) bool {
	if len(allowed) == 0 {
		return true
	}
	if slices.Contains(allowed, "*") {
		return true
	}
	return slices.Contains(allowed, value)
}

func normalize(value string) string {
	return strings.ToLower(strings.TrimSpace(strings.TrimPrefix(value, "/")))
}

func normalizeSlice(items []string) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		normalized := normalize(item)
		if normalized == "" {
			continue
		}
		out = append(out, normalized)
	}
	return out
}

func nonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}
