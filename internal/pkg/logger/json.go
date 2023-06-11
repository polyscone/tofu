package logger

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/polyscone/tofu/internal/pkg/errsx"
)

// JSONFormatter implements the Formatter interface for logging data as JSON.
type JSONFormatter struct{}

// Format implements the Formatter interface to be used in a log writer.
func (f *JSONFormatter) Format(message, newline string, at time.Time) string {
	var value any
	if err := json.Unmarshal([]byte(message), &value); err != nil {
		value = message
	}

	b := errsx.Must(json.Marshal(map[string]any{
		"at":  at.UTC().Format(time.RFC3339Nano),
		"msg": value,
	}))

	return fmt.Sprintf("%s%s", b, newline)
}
