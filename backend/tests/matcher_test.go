package tests

import (
	"testing"

	"github.com/runbook-hunter/runbook-hunter/backend/internal/runbooks"
)

func TestMatcherPicksRunbook(t *testing.T) {
	books := []runbooks.Definition{
		{Name: "first", Match: map[string]string{"service": "web"}},
		{Name: "second", Match: map[string]string{"service": "api", "env": "prod"}},
	}
	book, ok := runbooks.Match(map[string]string{"service": "api", "env": "prod"}, books)
	if !ok {
		t.Fatalf("expected match")
	}
	if book.Name != "second" {
		t.Fatalf("expected second, got %s", book.Name)
	}
}
