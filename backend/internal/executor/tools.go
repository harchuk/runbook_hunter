package executor

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

type CircuitBreaker struct {
	mu        sync.Mutex
	failures  map[string]int
	openUntil map[string]time.Time
	threshold int
	window    time.Duration
}

func NewCircuitBreaker(threshold int, window time.Duration) *CircuitBreaker {
	if threshold <= 0 {
		threshold = 3
	}
	if window <= 0 {
		window = time.Minute
	}
	return &CircuitBreaker{
		failures:  map[string]int{},
		openUntil: map[string]time.Time{},
		threshold: threshold,
		window:    window,
	}
}

func (cb *CircuitBreaker) Allow(key string, now time.Time) bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	until := cb.openUntil[key]
	return until.IsZero() || now.After(until)
}

func (cb *CircuitBreaker) Success(key string) {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	delete(cb.failures, key)
	delete(cb.openUntil, key)
}

func (cb *CircuitBreaker) Failure(key string, now time.Time) {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.failures[key]++
	if cb.failures[key] >= cb.threshold {
		cb.openUntil[key] = now.Add(cb.window)
		cb.failures[key] = 0
	}
}

// Security notes: allowlist is mandatory for network operations and blocks unknown egress targets.
func IsAllowedHost(rawURLOrHost string, allowlist []string) bool {
	host := rawURLOrHost
	if strings.Contains(rawURLOrHost, "://") {
		u, err := url.Parse(rawURLOrHost)
		if err != nil {
			return false
		}
		host = u.Hostname()
	} else {
		h, _, err := net.SplitHostPort(rawURLOrHost)
		if err == nil {
			host = h
		}
	}

	host = strings.ToLower(strings.TrimSpace(host))
	for _, allowed := range allowlist {
		allowed = strings.ToLower(strings.TrimSpace(allowed))
		if allowed == "" {
			continue
		}
		if strings.HasPrefix(allowed, "*.") {
			suffix := strings.TrimPrefix(allowed, "*")
			if strings.HasSuffix(host, suffix) {
				return true
			}
			continue
		}
		if host == allowed {
			return true
		}
	}
	return false
}

func HTTPGet(ctx context.Context, client *http.Client, rawURL string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return "", err
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 16*1024))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(body))), nil
}

func FetchJSON(ctx context.Context, client *http.Client, rawURL string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return "", err
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 16*1024))
	if err != nil {
		return "", err
	}
	var js any
	if err := json.Unmarshal(body, &js); err != nil {
		return "", fmt.Errorf("invalid json response: %w", err)
	}
	compact, _ := json.Marshal(js)
	return fmt.Sprintf("status=%d json=%s", resp.StatusCode, string(compact)), nil
}

func DNSLookup(host string) (string, error) {
	ips, err := net.LookupHost(host)
	if err != nil {
		return "", err
	}
	return strings.Join(ips, ","), nil
}

func TCPCheck(address string, timeout time.Duration) (string, error) {
	conn, err := net.DialTimeout("tcp", address, timeout)
	if err != nil {
		return "", err
	}
	_ = conn.Close()
	return "tcp_connect=ok", nil
}
