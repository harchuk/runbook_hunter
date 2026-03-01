package tests

import (
	"context"
	"testing"
	"time"
)

func TestDedupSuppression(t *testing.T) {
	repo := newTestRepo(t)
	mustPing(t, repo)
	ctx := context.Background()
	now := time.Now().UTC()

	send1, err := repo.CheckAndUpdateDedup(ctx, 1, "tg-default", "hash1", 5*time.Minute, false, now)
	if err != nil {
		t.Fatalf("dedup check 1: %v", err)
	}
	if !send1 {
		t.Fatalf("expected first send true")
	}

	send2, err := repo.CheckAndUpdateDedup(ctx, 1, "tg-default", "hash1", 5*time.Minute, false, now.Add(1*time.Minute))
	if err != nil {
		t.Fatalf("dedup check 2: %v", err)
	}
	if send2 {
		t.Fatalf("expected duplicate to be suppressed")
	}
}
