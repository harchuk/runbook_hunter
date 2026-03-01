package executor

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/runbook-hunter/runbook-hunter/backend/internal/runbooks"
)

type Result struct {
	StepName   string
	Tool       string
	Status     string
	Output     string
	Error      string
	StartedAt  time.Time
	FinishedAt time.Time
}

type Runner struct {
	Allowlist []string
	Timeout   time.Duration
	Retries   int
	Breaker   *CircuitBreaker
}

func NewRunner(allowlist []string, timeout time.Duration, retries int, breaker *CircuitBreaker) *Runner {
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	if retries < 0 {
		retries = 0
	}
	if breaker == nil {
		breaker = NewCircuitBreaker(3, time.Minute)
	}
	return &Runner{Allowlist: allowlist, Timeout: timeout, Retries: retries, Breaker: breaker}
}

func (r *Runner) Run(ctx context.Context, steps []runbooks.Step, maxSteps int) []Result {
	if maxSteps <= 0 || maxSteps > len(steps) {
		maxSteps = len(steps)
	}
	results := make([]Result, 0, maxSteps)
	for i := 0; i < maxSteps; i++ {
		results = append(results, r.executeStep(ctx, steps[i]))
	}
	return results
}

func (r *Runner) executeStep(ctx context.Context, step runbooks.Step) Result {
	res := Result{StepName: step.Name, Tool: step.Tool, StartedAt: time.Now().UTC()}
	if res.StepName == "" {
		res.StepName = step.Tool
	}

	var lastErr error
	for attempt := 0; attempt <= r.Retries; attempt++ {
		out, err := r.exec(ctx, step)
		if err == nil {
			res.Status = "ok"
			res.Output = out
			res.FinishedAt = time.Now().UTC()
			return res
		}
		lastErr = err
	}
	res.Status = "error"
	res.Error = lastErr.Error()
	res.FinishedAt = time.Now().UTC()
	return res
}

func (r *Runner) exec(ctx context.Context, step runbooks.Step) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, r.Timeout)
	defer cancel()

	client := &http.Client{Timeout: r.Timeout}
	switch step.Tool {
	case "http_get":
		rawURL := step.Args["url"]
		if !IsAllowedHost(rawURL, r.Allowlist) {
			return "", fmt.Errorf("host is not allowlisted")
		}
		host := hostKey(rawURL)
		if !r.Breaker.Allow(host, time.Now().UTC()) {
			return "", errors.New("circuit breaker open")
		}
		out, err := HTTPGet(ctx, client, rawURL)
		if err != nil {
			r.Breaker.Failure(host, time.Now().UTC())
			return "", err
		}
		r.Breaker.Success(host)
		return out, nil
	case "fetch_json":
		rawURL := step.Args["url"]
		if !IsAllowedHost(rawURL, r.Allowlist) {
			return "", fmt.Errorf("host is not allowlisted")
		}
		host := hostKey(rawURL)
		if !r.Breaker.Allow(host, time.Now().UTC()) {
			return "", errors.New("circuit breaker open")
		}
		out, err := FetchJSON(ctx, client, rawURL)
		if err != nil {
			r.Breaker.Failure(host, time.Now().UTC())
			return "", err
		}
		r.Breaker.Success(host)
		return out, nil
	case "dns_lookup":
		host := step.Args["host"]
		if !IsAllowedHost(host, r.Allowlist) {
			return "", fmt.Errorf("host is not allowlisted")
		}
		return DNSLookup(host)
	case "tcp_check":
		address := step.Args["address"]
		if !IsAllowedHost(address, r.Allowlist) {
			return "", fmt.Errorf("host is not allowlisted")
		}
		return TCPCheck(address, r.Timeout)
	default:
		return "", fmt.Errorf("unsupported read-only tool: %s", step.Tool)
	}
}

func hostKey(raw string) string {
	if idx := strings.Index(raw, "://"); idx >= 0 {
		trimmed := raw[idx+3:]
		if slash := strings.Index(trimmed, "/"); slash >= 0 {
			trimmed = trimmed[:slash]
		}
		return trimmed
	}
	return raw
}
