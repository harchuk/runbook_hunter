package reporter

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/runbook-hunter/runbook-hunter/backend/internal/config"
)

type MattermostReporter struct {
	Client *http.Client
}

func NewMattermostReporter(timeout time.Duration) *MattermostReporter {
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	return &MattermostReporter{Client: &http.Client{Timeout: timeout}}
}

func FormatMattermostMessage(data MessageData) string {
	return fmt.Sprintf(
		"### :rotating_light: [%s] %s\n**Service:** %s  |  **Env:** %s\n**Incident:** #%d  |  **Status:** %s  |  **Closure:** %s\n**Summary:** %s\n**Runbook:** %s\n**Criteria:** %s\n**Steps:** %s\n_Updated: %s_\n`Security:` read-only checks only",
		strings.ToUpper(nonEmpty(data.Severity, "unknown")),
		nonEmpty(data.AlertName, "unknown"),
		nonEmpty(data.Service, "n/a"),
		nonEmpty(data.Env, "n/a"),
		data.IncidentID,
		nonEmpty(data.Status, "open"),
		nonEmpty(data.ClosureState, "open"),
		summarizeText(data.Brief, 320),
		nonEmpty(data.RunbookName, "n/a"),
		summarizeText(data.ClosureSummary, 260),
		summarizeText(data.StepSummary, 260),
		data.UpdatedAt.UTC().Format(time.RFC3339),
	)
}

type mattermostWebhookRequest struct {
	Text    string `json:"text"`
	Channel string `json:"channel,omitempty"`
}

// Security notes: incoming webhook URL is secret and must not be logged.
func (r *MattermostReporter) Send(ctx context.Context, destination config.MattermostDestination, text string) error {
	if destination.WebhookURL == "" {
		return fmt.Errorf("mattermost webhook url is empty")
	}
	body, err := json.Marshal(mattermostWebhookRequest{Text: text, Channel: destination.Channel})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, destination.WebhookURL, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("mattermost request creation failed")
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := r.Client.Do(req)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return fmt.Errorf("mattermost request timed out")
		}
		return fmt.Errorf("mattermost request failed")
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("mattermost send failed with status %d", resp.StatusCode)
	}
	return nil
}
