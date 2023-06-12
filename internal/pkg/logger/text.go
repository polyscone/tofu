package logger

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// TextFormatter implements the Formatter interface for logging data in a
// pretty printed human readable style.
type TextFormatter struct{}

// Format implements the Formatter interface to be used in a log writer.
func (f *TextFormatter) Format(message, newline string, at time.Time) string {
	var value any
	if err := json.Unmarshal([]byte(message), &value); err == nil {
		b, err := json.MarshalIndent(value, "", "\t")
		if err == nil {
			message = string(b)
		}
	}

	message = strings.TrimRight(message, "\n\r\t")

	return fmt.Sprintf("[%v]\n%v\n\n", at.Format("15:04:05 MST"), message)
}
