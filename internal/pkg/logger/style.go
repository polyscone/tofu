package logger

import "errors"

const (
	Text Style = "text"
	JSON Style = "json"
)

var OutputStyle = JSON

type Style string

func (s *Style) isValid() bool {
	switch *s {
	case Text, JSON:
		return true
	}

	return false
}

func (s *Style) Set(value string) error {
	style := Style(value)
	if !style.isValid() {
		return errors.New(`style must be one of "text", or "json"`)
	}

	*s = style

	return nil
}

func (s Style) String() string {
	return string(s)
}
