package slogger

import (
	"context"
	"errors"
	"fmt"
	"io"
	"reflect"
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
	group string
	attrs []slog.Attr
	mu    *sync.Mutex
	w     io.Writer
}

func NewDevHandler(w io.Writer, level slog.Leveler) *DevHandler {
	if level == nil || reflect.TypeOf(level).Kind() == reflect.Ptr && reflect.ValueOf(level).IsNil() {
		level = &slog.LevelVar{}
	}

	return &DevHandler{
		level: level,
		mu:    new(sync.Mutex),
		w:     w,
	}
}

func (h *DevHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return level >= h.level.Level()
}

func (h *DevHandler) Handle(ctx context.Context, r slog.Record) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	var attrs string
	for _, a := range h.attrs {
		if !a.Equal(slog.Attr{}) {
			attrs += "  "

			if h.group != "" {
				attrs += h.group + "."
			}

			attrs += a.Key + ": " + a.Value.String() + "\n"
		}
	}

	r.Attrs(func(a slog.Attr) bool {
		if !a.Equal(slog.Attr{}) {
			attrs += "  "

			if h.group != "" {
				attrs += h.group + "."
			}

			attrs += a.Key + ": " + a.Value.String() + "\n"
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

func (h *DevHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &DevHandler{
		level: h.level,
		group: h.group,
		attrs: append(h.attrs, attrs...),
		mu:    h.mu,
		w:     h.w,
	}
}

func (h *DevHandler) WithGroup(name string) slog.Handler {
	return &DevHandler{
		level: h.level,
		group: strings.TrimSuffix(name+"."+h.group, "."),
		attrs: h.attrs,
		mu:    h.mu,
		w:     h.w,
	}
}
