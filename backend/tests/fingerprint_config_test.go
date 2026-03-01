package tests

import (
	"testing"

	"github.com/runbook-hunter/runbook-hunter/backend/internal/correlation"
)

func TestFingerprintDefaultDeterministic(t *testing.T) {
	labels := map[string]string{
		"alertname": "A",
		"service":   "svc",
		"env":       "prod",
		"instance":  "i-1",
	}
	fp1, _ := correlation.Build(labels, "route-1", nil)
	fp2, _ := correlation.Build(labels, "route-1", nil)
	if fp1 != fp2 {
		t.Fatalf("expected deterministic fingerprints")
	}
}

func TestFingerprintCustomFields(t *testing.T) {
	labels := map[string]string{
		"alertname": "A",
		"service":   "svc",
		"env":       "prod",
		"job":       "j-1",
	}
	fpDefault, _ := correlation.Build(labels, "route-1", nil)
	fpCustom, _ := correlation.Build(labels, "route-1", []string{"alertname", "job"})
	if fpDefault == fpCustom {
		t.Fatalf("expected different fingerprint for custom fields")
	}
}
