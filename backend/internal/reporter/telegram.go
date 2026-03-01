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
	return fmt.Sprintf(
		"🚨 [%s] %s\nService: %s | Env: %s\nIncident: #%d | Status: %s\nSummary: %s\nRunbook: %s\nUpdated: %s",
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
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := r.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("telegram send failed with status %d", resp.StatusCode)
	}
	return nil
}
