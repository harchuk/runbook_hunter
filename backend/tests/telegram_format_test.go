package tests

import (
	"strings"
	"testing"
	"time"

	"github.com/runbook-hunter/runbook-hunter/backend/internal/reporter"
)

func TestTelegramFormatting(t *testing.T) {
	msg := reporter.FormatTelegramMessage(reporter.MessageData{
		Severity:    "critical",
		AlertName:   "API5xxSpike",
		Service:     "api",
		Env:         "prod",
		IncidentID:  42,
		Status:      "open",
		Brief:       "facts",
		RunbookName: "API 5xx spike",
		UpdatedAt:   time.Unix(0, 0).UTC(),
	})
	if !strings.Contains(msg, "[CRITICAL]") || !strings.Contains(msg, "Incident: #42") || !strings.Contains(msg, "Security: read-only checks") {
		t.Fatalf("unexpected telegram format: %s", msg)
	}
}
