package slogger

import (
	"fmt"
	"os"

	"golang.org/x/exp/slog"
)

func New(style Style, level *slog.LevelVar) (*slog.Logger, error) {
	if level == nil {
		level = &slog.LevelVar{}
	}

	handler, err := NewHandler(style, level)
	if err != nil {
		return nil, fmt.Errorf("new handler: %w", err)
	}

	return slog.New(handler), nil
}

func NewHandler(style Style, level *slog.LevelVar) (slog.Handler, error) {
	switch style {
	case StyleJSON:
		return slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level}), nil

	case StyleText:
		return slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: level}), nil

	case StyleDev:
		return NewDevHandler(os.Stdout, level), nil

	default:
		return nil, fmt.Errorf("unknown log style %q", style)
	}
}
