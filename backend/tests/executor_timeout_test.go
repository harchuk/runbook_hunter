package tests

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/runbook-hunter/runbook-hunter/backend/internal/executor"
	"github.com/runbook-hunter/runbook-hunter/backend/internal/runbooks"
)

func TestExecutorTimeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	defer srv.Close()

	runner := executor.NewRunner([]string{"127.0.0.1", "localhost"}, 50*time.Millisecond, 0, nil)
	results := runner.Run(context.Background(), []runbooks.Step{{
		Name: "slow",
		Tool: "http_get",
		Args: map[string]string{"url": srv.URL},
	}}, 1)
	if results[0].Status != "error" {
		t.Fatalf("expected timeout error, got %s", results[0].Status)
	}
}
