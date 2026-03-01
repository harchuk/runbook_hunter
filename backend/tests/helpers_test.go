package tests

import (
	"context"
	"encoding/base64"
	"testing"

	"github.com/runbook-hunter/runbook-hunter/backend/internal/config"
	"github.com/runbook-hunter/runbook-hunter/backend/internal/store"
)

func newTestRepo(t *testing.T) *store.Repository {
	t.Helper()
	repo, err := store.Open("sqlite", "file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("open repo: %v", err)
	}
	if err := repo.EnsureLocalSchema(); err != nil {
		t.Fatalf("ensure schema: %v", err)
	}
	return repo
}

func baseConfig() config.Config {
	cfg := config.BuiltInDefaults()
	cfg.Database.Driver = "sqlite"
	cfg.Database.DSN = "file::memory:?cache=shared"
	cfg.Auth.BasicUser = "admin"
	cfg.Auth.BasicPass = "password"
	cfg.Security.SettingsCryptoKeyB64 = base64.StdEncoding.EncodeToString([]byte("0123456789abcdef0123456789abcdef"))
	return cfg
}

func mustPing(t *testing.T, repo *store.Repository) {
	t.Helper()
	if err := repo.Ping(context.Background()); err != nil {
		t.Fatalf("ping: %v", err)
	}
}
