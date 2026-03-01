package tests

import (
	"context"
	"testing"
	"time"

	"github.com/runbook-hunter/runbook-hunter/backend/internal/executor"
	"github.com/runbook-hunter/runbook-hunter/backend/internal/runbooks"
)

func TestExecutorAllowlistDeny(t *testing.T) {
	r := executor.NewRunner([]string{"allowed.local"}, time.Second, 0, nil)
	results := r.Run(context.Background(), []runbooks.Step{{
		Name: "deny",
		Tool: "http_get",
		Args: map[string]string{"url": "http://denied.local/healthz"},
	}}, 1)
	if len(results) != 1 || results[0].Status != "error" {
		t.Fatalf("expected denied step to fail")
	}
}
