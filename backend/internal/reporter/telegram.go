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

type MessageData struct {
	Severity    string
	AlertName   string
	Service     string
	Env         string
	IncidentID  uint
	Status      string
	Brief       string
	RunbookName string
	UpdatedAt   time.Time
}

type TelegramReporter struct {
	Client *http.Client
}

func NewTelegramReporter(timeout time.Duration) *TelegramReporter {
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	return &TelegramReporter{Client: &http.Client{Timeout: timeout}}
}

func FormatTelegramMessage(data MessageData) string {
	severity := strings.ToUpper(nonEmpty(data.Severity, "unknown"))
	status := strings.ToUpper(nonEmpty(data.Status, "open"))
	findings := summarizeBrief(data.Brief, 5)
	updatedAt := data.UpdatedAt.UTC().Format("2006-01-02 15:04:05 MST")

	parts := []string{
		fmt.Sprintf("%s Runbook Hunter", severityIcon(severity)),
		fmt.Sprintf("%s %s  |  %s %s", severityIcon(severity), severity, statusIcon(status), status),
		fmt.Sprintf("Alert: %s", nonEmpty(data.AlertName, "n/a")),
		fmt.Sprintf("Scope: service=%s env=%s", nonEmpty(data.Service, "n/a"), nonEmpty(data.Env, "n/a")),
		fmt.Sprintf("Incident: #%d", data.IncidentID),
		fmt.Sprintf("Runbook: %s", nonEmpty(data.RunbookName, "n/a")),
		"Findings:",
		findings,
		fmt.Sprintf("Updated: %s", updatedAt),
		"Mode: read-only diagnostics",
	}
	return strings.Join(parts, "\n")
}

type telegramRequest struct {
	ChatID          string `json:"chat_id"`
	Text            string `json:"text"`
	MessageThreadID int64  `json:"message_thread_id,omitempty"`
	ParseMode       string `json:"parse_mode,omitempty"`
}

// Security notes: token is accepted only from effective config/secret and never logged.
func (r *TelegramReporter) Send(ctx context.Context, destination config.TelegramDestination, text string) error {
	if destination.BotToken == "" {
		return fmt.Errorf("telegram bot token is empty")
	}
	if destination.ChatID == "" {
		return fmt.Errorf("telegram chat id is empty")
	}
	body, err := json.Marshal(telegramRequest{
		ChatID:          destination.ChatID,
		Text:            text,
		MessageThreadID: destination.TopicID,
	})
	if err != nil {
		return err
	}
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", destination.BotToken)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("telegram request creation failed")
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := r.Client.Do(req)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return fmt.Errorf("telegram request timed out")
		}
		return fmt.Errorf("telegram request failed")
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("telegram send failed with status %d", resp.StatusCode)
	}
	return nil
}

func nonEmpty(value, fallback string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return fallback
	}
	return trimmed
}

func severityIcon(severity string) string {
	switch strings.ToLower(strings.TrimSpace(severity)) {
	case "critical":
		return "🚨"
	case "warning":
		return "⚠️"
	case "info":
		return "ℹ️"
	default:
		return "🛰️"
	}
}

func statusIcon(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "resolved":
		return "✅"
	case "open", "firing":
		return "🟠"
	default:
		return "🔎"
	}
}

func summarizeBrief(brief string, maxLines int) string {
	brief = strings.TrimSpace(brief)
	if brief == "" {
		return "- no facts yet"
	}
	chunks := strings.Split(brief, ";")
	lines := make([]string, 0, len(chunks))
	for _, chunk := range chunks {
		chunk = strings.TrimSpace(chunk)
		if chunk == "" {
			continue
		}
		if len(chunk) > 180 {
			chunk = chunk[:177] + "..."
		}
		lines = append(lines, "- "+chunk)
		if len(lines) >= maxLines {
			break
		}
	}
	if len(lines) == 0 {
		cut := brief
		if len(cut) > 180 {
			cut = cut[:177] + "..."
		}
		return "- " + cut
	}
	if len(chunks) > len(lines) {
		lines = append(lines, fmt.Sprintf("- ... and %d more facts", len(chunks)-len(lines)))
	}
	return strings.Join(lines, "\n")
}
