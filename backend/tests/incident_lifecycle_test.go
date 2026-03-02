package tests

import (
	"context"
	"testing"
	"time"

	"github.com/runbook-hunter/runbook-hunter/backend/internal/store"
)

func TestIncidentAggregationSingleFingerprint(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()
	now := time.Now().UTC()

	first, err := repo.UpsertIncidentAndSignal(ctx, store.AlertSignalInput{
		Fingerprint: "fp-1",
		RouteKey:    "default",
		AlertName:   "VMHighCPU",
		Status:      "firing",
		Labels: map[string]string{
			"alertname": "VMHighCPU",
			"service":   "vm",
			"env":       "prod",
			"instance":  "vm-1",
		},
		Annotations: map[string]string{"summary": "cpu high"},
		StartsAt:    now,
	})
	if err != nil {
		t.Fatalf("upsert first: %v", err)
	}
	second, err := repo.UpsertIncidentAndSignal(ctx, store.AlertSignalInput{
		Fingerprint: "fp-1",
		RouteKey:    "default",
		AlertName:   "VMHighCPU",
		Status:      "firing",
		Labels: map[string]string{
			"alertname": "VMHighCPU",
			"service":   "vm",
			"env":       "prod",
			"instance":  "vm-1",
		},
		Annotations: map[string]string{"summary": "cpu still high"},
		StartsAt:    now.Add(1 * time.Minute),
	})
	if err != nil {
		t.Fatalf("upsert second: %v", err)
	}
	if first.ID != second.ID {
		t.Fatalf("expected same incident id, got %d and %d", first.ID, second.ID)
	}
}

func TestIncidentAlertStatsTracksResolvedState(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()
	now := time.Now().UTC()

	inc, err := repo.UpsertIncidentAndSignal(ctx, store.AlertSignalInput{
		Fingerprint: "fp-2",
		RouteKey:    "default",
		AlertName:   "VMHighCPU",
		Status:      "firing",
		Labels: map[string]string{
			"alertname": "VMHighCPU",
			"service":   "vm",
			"env":       "prod",
			"instance":  "vm-2",
		},
		StartsAt: now,
	})
	if err != nil {
		t.Fatalf("upsert firing: %v", err)
	}
	if _, err := repo.UpsertIncidentAndSignal(ctx, store.AlertSignalInput{
		Fingerprint: "fp-2",
		RouteKey:    "default",
		AlertName:   "VMHighCPU",
		Status:      "resolved",
		Labels: map[string]string{
			"alertname": "VMHighCPU",
			"service":   "vm",
			"env":       "prod",
			"instance":  "vm-2",
		},
		StartsAt: now,
		EndsAt:   ptrTime(now.Add(3 * time.Minute)),
	}); err != nil {
		t.Fatalf("upsert resolved: %v", err)
	}
	total, firing, err := repo.IncidentAlertStats(ctx, inc.ID)
	if err != nil {
		t.Fatalf("incident stats: %v", err)
	}
	if total != 1 || firing != 0 {
		t.Fatalf("expected total=1 firing=0, got total=%d firing=%d", total, firing)
	}
}

func TestRunbookExecutionDedupSingleActive(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()
	now := time.Now().UTC()

	inc, err := repo.UpsertIncidentAndSignal(ctx, store.AlertSignalInput{
		Fingerprint: "fp-rb",
		RouteKey:    "default",
		AlertName:   "VMHighCPU",
		Status:      "firing",
		Labels: map[string]string{
			"alertname": "VMHighCPU",
			"service":   "vm",
			"env":       "prod",
			"instance":  "vm-rb",
		},
		StartsAt: now,
	})
	if err != nil {
		t.Fatalf("create incident for execution: %v", err)
	}

	plan := []store.ExecutionStepInput{{StepName: "step-1", StepOrder: 1, Kind: "check", Status: "pending"}}
	first, created, err := repo.EnsureRunbookExecution(ctx, inc.ID, "rb", "hash-v1", "auto", plan)
	if err != nil {
		t.Fatalf("ensure execution 1: %v", err)
	}
	if !created {
		t.Fatalf("expected first execution to be created")
	}
	second, created, err := repo.EnsureRunbookExecution(ctx, inc.ID, "rb", "hash-v1", "auto", plan)
	if err != nil {
		t.Fatalf("ensure execution 2: %v", err)
	}
	if created {
		t.Fatalf("expected second ensure to reuse active execution")
	}
	if first.ID != second.ID {
		t.Fatalf("expected same execution id, got %d and %d", first.ID, second.ID)
	}
}

func TestApprovalLifecyclePendingApproved(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()
	now := time.Now().UTC()

	inc, err := repo.UpsertIncidentAndSignal(ctx, store.AlertSignalInput{
		Fingerprint: "fp-approval",
		RouteKey:    "default",
		AlertName:   "DBPoolExhausted",
		Status:      "firing",
		Labels: map[string]string{
			"alertname": "DBPoolExhausted",
			"service":   "db",
			"env":       "prod",
			"instance":  "db-1",
		},
		StartsAt: now,
	})
	if err != nil {
		t.Fatalf("create incident for approval: %v", err)
	}
	exec, _, err := repo.EnsureRunbookExecution(ctx, inc.ID, "rb", "hash-v1", "auto", []store.ExecutionStepInput{
		{StepName: "ansible", StepOrder: 1, Kind: "action", Status: "pending"},
	})
	if err != nil {
		t.Fatalf("create execution for approval: %v", err)
	}

	req, err := repo.GetOrCreateApproval(ctx, inc.ID, exec.ID, 1, "ansible_awx_job", "needs approval")
	if err != nil {
		t.Fatalf("create approval: %v", err)
	}
	if req.Status != "pending" {
		t.Fatalf("expected pending, got %s", req.Status)
	}
	if err := repo.UpdateApprovalStatus(ctx, req.ID, "approved", "approved in test"); err != nil {
		t.Fatalf("approve request: %v", err)
	}
	updated, err := repo.GetApproval(ctx, req.ID)
	if err != nil {
		t.Fatalf("get approval: %v", err)
	}
	if updated.Status != "approved" {
		t.Fatalf("expected approved status, got %s", updated.Status)
	}
}

func ptrTime(v time.Time) *time.Time {
	return &v
}
