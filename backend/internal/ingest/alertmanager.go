package ingest

import (
	"encoding/json"
	"errors"
	"io"
	"time"
)

type Webhook struct {
	Version           string            `json:"version"`
	GroupKey          string            `json:"groupKey"`
	TruncatedAlerts   int               `json:"truncatedAlerts"`
	Status            string            `json:"status"`
	Receiver          string            `json:"receiver"`
	GroupLabels       map[string]string `json:"groupLabels"`
	CommonLabels      map[string]string `json:"commonLabels"`
	CommonAnnotations map[string]string `json:"commonAnnotations"`
	ExternalURL       string            `json:"externalURL"`
	Alerts            []Alert           `json:"alerts"`
}

type Alert struct {
	Status       string            `json:"status"`
	Labels       map[string]string `json:"labels"`
	Annotations  map[string]string `json:"annotations"`
	StartsAt     time.Time         `json:"startsAt"`
	EndsAt       *time.Time        `json:"endsAt,omitempty"`
	GeneratorURL string            `json:"generatorURL"`
	Fingerprint  string            `json:"fingerprint,omitempty"`
}

func ParseAlertmanagerPayload(r io.Reader) (Webhook, error) {
	dec := json.NewDecoder(r)
	dec.DisallowUnknownFields()
	var payload Webhook
	if err := dec.Decode(&payload); err != nil {
		return Webhook{}, err
	}
	if len(payload.Alerts) == 0 {
		return Webhook{}, errors.New("alerts array is required")
	}
	for _, a := range payload.Alerts {
		if a.StartsAt.IsZero() {
			return Webhook{}, errors.New("alert startsAt is required")
		}
		if len(a.Labels) == 0 {
			return Webhook{}, errors.New("alert labels are required")
		}
	}
	return payload, nil
}
