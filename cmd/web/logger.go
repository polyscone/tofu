package main

import "errors"

const (
	styleText LoggerStyle = "text"
	styleJSON LoggerStyle = "json"
)

type LoggerStyle string

func (s *LoggerStyle) isValid() bool {
	switch *s {
	case styleText, styleJSON:
		return true
	}

	return false
}

func (s *LoggerStyle) Set(value string) error {
	style := LoggerStyle(value)
	if !style.isValid() {
		return errors.New(`style must be one of "text", or "json"`)
	}

	*s = style

	return nil
}

func (s LoggerStyle) String() string {
	return string(s)
}
