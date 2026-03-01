package tests

import (
	"testing"

	"github.com/runbook-hunter/runbook-hunter/backend/internal/config"
	"github.com/runbook-hunter/runbook-hunter/backend/internal/router"
)

func TestRoutingSelection(t *testing.T) {
	cfg := config.BuiltInDefaults()
	cfg.Destinations.Telegram = []config.TelegramDestination{{ID: "tg", Enabled: true, ChatID: "1", BotToken: "x"}}
	cfg.Destinations.Mattermost = []config.MattermostDestination{{ID: "mm", Enabled: true, WebhookURL: "https://example.com"}}
	cfg.Routing.Rules = []config.RoutingRule{{
		Name:         "critical",
		Enabled:      true,
		Priority:     1,
		MatchLabels:  map[string]string{"severity": "critical"},
		Destinations: []string{"tg", "mm"},
	}}
	eng := router.New()
	dests := eng.SelectDestinations(map[string]string{"severity": "critical"}, cfg, nil)
	if len(dests) != 2 {
		t.Fatalf("expected 2 destinations, got %d", len(dests))
	}
}
