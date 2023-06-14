package logger

import (
	"encoding/json"
	"fmt"
	"time"
)

// JSONFormatter implements the Formatter interface for logging data as JSON.
type JSONFormatter struct{}

// Format implements the Formatter interface to be used in a log writer.
func (f *JSONFormatter) Format(message, newline string, at time.Time) string {
	var value map[string]any
	if err := json.Unmarshal([]byte(message), &value); err != nil {
		value = map[string]any{"msg": message}
	}

	atFormatted := at.UTC().Format(time.RFC3339Nano)

	value["at"] = atFormatted

	b, err := json.Marshal(value)
	if err != nil {
		return fmt.Sprintf(`{"at":%q"msg":%q}%v`, atFormatted, err, newline)
	}

	return fmt.Sprintf("%s%v", b, newline)
}
