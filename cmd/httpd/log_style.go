package main

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
)

type LogStyle string

func (l LogStyle) isValid() bool {
	return l == "text" || l == "json"
}

func (l LogStyle) NewHandler(level *slog.LevelVar) (slog.Handler, error) {
	if level == nil {
		level = &slog.LevelVar{}
	}

	switch l {
	case "json":
		return slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level}), nil

	case "text":
		return slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: level}), nil

	default:
		return nil, fmt.Errorf("unknown log style %q", l)
	}
}

func (l LogStyle) NewLogger(level *slog.LevelVar) (*slog.Logger, error) {
	handler, err := l.NewHandler(level)
	if err != nil {
		return nil, fmt.Errorf("new handler: %w", err)
	}

	return slog.New(handler), nil
}

func (l *LogStyle) Set(value string) error {
	style := LogStyle(value)
	if !style.isValid() {
		return errors.New(`log style must be "text" or "json"`)
	}

	*l = style

	return nil
}

func (l LogStyle) String() string {
	return string(l)
}
