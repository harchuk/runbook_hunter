package reporter

import (
	"bytes"
	"context"
	"encoding/json"
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
		"### :rotating_light: [%s] %s\n**Service:** %s  |  **Env:** %s\n**Incident:** #%d  |  **Status:** %s\n**Summary:** %s\n**Runbook:** %s\n_Updated: %s_",
		strings.ToUpper(data.Severity),
		data.AlertName,
		data.Service,
		data.Env,
		data.IncidentID,
		data.Status,
		data.Brief,
		data.RunbookName,
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
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := r.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("mattermost send failed with status %d", resp.StatusCode)
	}
	return nil
}
