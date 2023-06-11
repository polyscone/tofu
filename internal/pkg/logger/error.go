package logger

import (
	"encoding/json"
	"fmt"
)

// SprintErrorf returns a pretty printed string of an error in the
// style set in OutputStyle.
func SprintErrorf(format any, a ...any) string {
	var err error
	switch format := format.(type) {
	case error:
		err = format

	case string:
		err = fmt.Errorf(format, a...)

	default:
		panic("want error or string message")
	}

	if OutputStyle == JSON {
		return SprintJSON(err)
	}

	return Sprint(err)
}

// PrintErrorf pretty prints an error in the style set in OutputStyle.
func PrintErrorf(format any, a ...any) {
	Error.Print(SprintErrorf(format, a...))
}

// Sprint returns the given error's string value by calling Error(), unless it
// is a Trace, in which case it will return String().
func Sprint(err error) string {
	if err == nil {
		return "<nil>"
	}

	return err.Error()
}

// SprintJSON returns the given error as a JSON string.
func SprintJSON(err error) string {
	b, err := json.Marshal(err)
	if err != nil {
		b = []byte(err.Error())
	}

	return string(b)
}
