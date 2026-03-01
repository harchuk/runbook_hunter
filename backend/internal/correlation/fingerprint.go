package correlation

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

var defaultFields = []string{"alertname", "service", "env", "instance|job"}

func Build(labels map[string]string, routeKey string, fields []string) (string, string) {
	if len(fields) == 0 {
		fields = defaultFields
	}
	parts := make([]string, 0, len(fields)+1)
	for _, field := range fields {
		if strings.Contains(field, "|") {
			parts = append(parts, firstPresent(labels, strings.Split(field, "|")...))
			continue
		}
		parts = append(parts, strings.TrimSpace(labels[field]))
	}
	parts = append(parts, strings.TrimSpace(routeKey))
	joined := strings.Join(parts, "|")
	h := sha256.Sum256([]byte(joined))
	return hex.EncodeToString(h[:]), joined
}

func firstPresent(labels map[string]string, fields ...string) string {
	for _, f := range fields {
		if value := strings.TrimSpace(labels[strings.TrimSpace(f)]); value != "" {
			return value
		}
	}
	return ""
}
