package tests

import (
	"strings"
	"testing"

	"github.com/runbook-hunter/runbook-hunter/backend/internal/ingest"
)

func TestParseAlertmanagerPayloadValid(t *testing.T) {
	jsonPayload := `{
	  "version":"4",
	  "groupKey":"{}:{alertname=\"API5xxSpike\"}",
	  "status":"firing",
	  "receiver":"runbook-hunter",
	  "alerts":[{
	    "status":"firing",
	    "labels":{"alertname":"API5xxSpike","service":"api","env":"prod"},
	    "annotations":{"summary":"x"},
	    "startsAt":"2026-03-01T10:00:00Z",
	    "generatorURL":"http://prometheus"
	  }]
	}`
	payload, err := ingest.ParseAlertmanagerPayload(strings.NewReader(jsonPayload))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(payload.Alerts) != 1 {
		t.Fatalf("expected 1 alert, got %d", len(payload.Alerts))
	}
}

func TestParseAlertmanagerPayloadInvalid(t *testing.T) {
	jsonPayload := `{"alerts":[{"status":"firing","labels":{},"startsAt":"2026-03-01T10:00:00Z"}]}`
	_, err := ingest.ParseAlertmanagerPayload(strings.NewReader(jsonPayload))
	if err == nil {
		t.Fatal("expected error")
	}
}
