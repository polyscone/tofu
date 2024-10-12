package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"runtime"
	"strings"
	"sync"
)

type LogStyle string

func (l LogStyle) isValid() bool {
	return l == "text" || l == "json" || l == "dev"
}

func (l LogStyle) NewHandler(level *slog.LevelVar) (slog.Handler, error) {
	if level == nil {
		level = &slog.LevelVar{}
	}

	opts := &slog.HandlerOptions{
		AddSource: true,
		Level:     level,
	}

	switch l {
	case "json":
		return slog.NewJSONHandler(os.Stdout, opts), nil

	case "text":
		return slog.NewTextHandler(os.Stdout, opts), nil

	case "dev":
		return NewDevHandler(os.Stdout, opts), nil

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

type DevHandler struct {
	opts  *slog.HandlerOptions
	group string
	attrs []slog.Attr
	mu    *sync.Mutex
	w     io.Writer
}

func NewDevHandler(w io.Writer, opts *slog.HandlerOptions) *DevHandler {
	if opts == nil {
		opts = &slog.HandlerOptions{}
	}

	return &DevHandler{
		opts: opts,
		mu:   new(sync.Mutex),
		w:    w,
	}
}

func (h *DevHandler) Enabled(ctx context.Context, level slog.Level) bool {
	minLevel := slog.LevelInfo
	if h.opts.Level != nil {
		minLevel = h.opts.Level.Level()
	}

	return level >= minLevel
}

func (h *DevHandler) Handle(ctx context.Context, r slog.Record) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	var attrs string
	for _, a := range h.attrs {
		if !a.Equal(slog.Attr{}) {
			if h.group != "" {
				attrs += h.group + "."
			}

			attrs += a.Key + ": " + a.Value.String() + "\n"
		}
	}

	r.Attrs(func(a slog.Attr) bool {
		if !a.Equal(slog.Attr{}) {
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

	var source string
	if h.opts.AddSource {
		fs := runtime.CallersFrames([]uintptr{r.PC})
		f, _ := fs.Next()

		source += fmt.Sprintf("\nfunction: %v", f.Function)
		source += fmt.Sprintf("\nsource: %v:%v", f.File, f.Line)
	}

	fmt.Fprintf(h.w, "[%v]\nmessage: %v\n%v%v%v", r.Time.Format("15:04:05 MST"), r.Message, attrs, source, newlines)

	return nil
}

func (h *DevHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &DevHandler{
		opts:  h.opts,
		group: h.group,
		attrs: append(h.attrs, attrs...),
		mu:    h.mu,
		w:     h.w,
	}
}

func (h *DevHandler) WithGroup(name string) slog.Handler {
	return &DevHandler{
		opts:  h.opts,
		group: strings.TrimSuffix(name+"."+h.group, "."),
		attrs: h.attrs,
		mu:    h.mu,
		w:     h.w,
	}
}
