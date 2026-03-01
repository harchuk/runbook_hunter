package obs

import (
	"io"
	"os"
	"strings"

	"github.com/rs/zerolog"
)

var sensitiveKeys = []string{"token", "webhook", "authorization", "password", "secret"}

func NewLogger(level string, out io.Writer) zerolog.Logger {
	if out == nil {
		out = os.Stdout
	}
	parsed, err := zerolog.ParseLevel(strings.ToLower(level))
	if err != nil {
		parsed = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(parsed)
	return zerolog.New(out).With().Timestamp().Logger()
}

// Security notes: use this helper before writing dynamic key/value payloads to logs.
func RedactMap(input map[string]any) map[string]any {
	output := make(map[string]any, len(input))
	for key, value := range input {
		if isSensitive(key) {
			output[key] = "***REDACTED***"
			continue
		}
		output[key] = value
	}
	return output
}

func isSensitive(key string) bool {
	lower := strings.ToLower(key)
	for _, sensitive := range sensitiveKeys {
		if strings.Contains(lower, sensitive) {
			return true
		}
	}
	return false
}
