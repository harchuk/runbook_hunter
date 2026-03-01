package tests

import (
	"context"
	"testing"

	"github.com/runbook-hunter/runbook-hunter/backend/internal/settings"
)

func TestSettingsPrecedence(t *testing.T) {
	repo := newTestRepo(t)
	cfg := baseConfig()
	cfg.Server.Addr = ":8080"

	svc, err := settings.NewService(repo, cfg, cfg.Security.SettingsCryptoKeyB64)
	if err != nil {
		t.Fatalf("new settings service: %v", err)
	}

	err = svc.PutOverrides(context.Background(), map[string]any{
		"server": map[string]any{"addr": ":9090"},
	})
	if err != nil {
		t.Fatalf("put overrides: %v", err)
	}

	effective, err := svc.EffectiveConfig(context.Background())
	if err != nil {
		t.Fatalf("effective config: %v", err)
	}
	if effective.Server.Addr != ":9090" {
		t.Fatalf("expected override addr :9090, got %s", effective.Server.Addr)
	}
}
