package slogger

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"

	"golang.org/x/exp/slog"
)

const (
	StyleText Style = "text"
	StyleJSON Style = "json"
	StyleDev  Style = "dev"
)

type Style string

func (s *Style) isValid() bool {
	switch *s {
	case StyleText, StyleJSON, StyleDev:
		return true
	}

	return false
}

func (s *Style) Set(value string) error {
	style := Style(value)
	if !style.isValid() {
		return errors.New(`style must be one of "text", "json", or "dev"`)
	}

	*s = style

	return nil
}

func (s Style) String() string {
	return string(s)
}

type DevHandler struct {
	level slog.Leveler
	mu    sync.Mutex
	w     io.Writer
}

func NewDevHandler(w io.Writer, level slog.Leveler) *DevHandler {
	return &DevHandler{
		level: level,
		w:     w,
	}
}

func (h *DevHandler) Enabled(ctx context.Context, level slog.Level) bool {
	minLevel := slog.LevelInfo
	if h.level != nil {
		minLevel = h.level.Level()
	}

	return level >= minLevel
}

// TODO:
// - If r.Time is the zero time, ignore the time
// - If r.PC is zero, ignore it
// - Attr's values should be resolved
// - If a group's key is empty, inline the group's Attrs
// - If a group has no Attrs (even if it has a non-empty key), ignore it
func (h *DevHandler) Handle(ctx context.Context, r slog.Record) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	var attrs string
	r.Attrs(func(a slog.Attr) bool {
		if !a.Equal(slog.Attr{}) {
			attrs += "  " + a.Key + ": " + a.Value.String() + "\n"
		}

		return true
	})

	attrs = strings.TrimRight(attrs, "\n")

	var newlines string
	if attrs != "" {
		newlines = "\n\n"
	}

	fmt.Fprintf(h.w, "[%v] %v\n%v%v", r.Time.Format("15:04:05 MST"), r.Message, attrs, newlines)

	return nil
}

// TODO: Implement creating a new handler with attrs
func (h *DevHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return h
}

// TODO: Implement creating a new handler with group name
func (h *DevHandler) WithGroup(name string) slog.Handler {
	return h
}
