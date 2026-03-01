package tests

import (
	"strings"
	"testing"
	"time"

	"github.com/runbook-hunter/runbook-hunter/backend/internal/reporter"
)

func TestMattermostFormatting(t *testing.T) {
	msg := reporter.FormatMattermostMessage(reporter.MessageData{
		Severity:    "warning",
		AlertName:   "LatencyP95Spike",
		Service:     "api",
		Env:         "prod",
		IncidentID:  7,
		Status:      "open",
		Brief:       "facts",
		RunbookName: "Latency p95 spike",
		UpdatedAt:   time.Unix(0, 0).UTC(),
	})
	if !strings.Contains(msg, "### :rotating_light:") || !strings.Contains(msg, "**Incident:** #7") {
		t.Fatalf("unexpected mattermost format: %s", msg)
	}
}
