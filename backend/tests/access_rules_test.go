package tests

import (
	"testing"

	"github.com/runbook-hunter/runbook-hunter/backend/internal/authn"
	"github.com/runbook-hunter/runbook-hunter/backend/internal/authz"
	"github.com/runbook-hunter/runbook-hunter/backend/internal/config"
	"github.com/runbook-hunter/runbook-hunter/backend/internal/store"
)

func TestAccessRulesFilterIncidentByGroup(t *testing.T) {
	engine := authz.NewEngine(config.AccessConfig{
		GroupRules: []config.GroupRule{
			{
				Group:      "noc-prod",
				Services:   []string{"api"},
				Envs:       []string{"prod"},
				Severities: []string{"critical"},
			},
		},
	})
	principal := authn.Principal{Groups: []string{"noc-prod"}, IsAdmin: false}
	allowed := store.Incident{Service: "api", Env: "prod", Severity: "critical", AlertName: "API5xxSpike"}
	denied := store.Incident{Service: "api", Env: "dev", Severity: "critical", AlertName: "API5xxSpike"}
	if !engine.CanViewIncident(principal, allowed) {
		t.Fatalf("expected incident to be visible for matching group rule")
	}
	if engine.CanViewIncident(principal, denied) {
		t.Fatalf("expected incident to be hidden for non-matching env")
	}
}

func TestAccessRulesAdminBypass(t *testing.T) {
	engine := authz.NewEngine(config.AccessConfig{
		GroupRules: []config.GroupRule{
			{Group: "noc-prod", Services: []string{"api"}, Envs: []string{"prod"}},
		},
	})
	principal := authn.Principal{Groups: []string{"other"}, IsAdmin: true}
	incident := store.Incident{Service: "db", Env: "stage", Severity: "warning", AlertName: "DBPoolExhausted"}
	if !engine.CanViewIncident(principal, incident) {
		t.Fatalf("expected admin principal to bypass group rules")
	}
}
