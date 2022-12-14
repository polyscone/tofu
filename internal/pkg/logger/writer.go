package logger

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"github.com/polyscone/tofu/internal/pkg/errors"
)

var (
	exedir = filepath.ToSlash(filepath.Dir(errors.Must(os.Executable())))
	info   = errors.MustOK(debug.ReadBuildInfo())
)

// Formatter represents a formatter that can be used by a
// logger to format messages.
type Formatter interface {
	Format(message, newline string, at time.Time, funcName, file string, line int) string
}

// Writer implements a simple log writer that can output
// logs in different styles.
type Writer struct {
	mu sync.Mutex
	w  io.Writer
	f  Formatter
}

// New returns a new log writer that will output logs in the given style.
func New(w io.Writer, style Style) *Writer {
	var f Formatter
	switch style {
	case Text:
		f = &TextFormatter{}

	case JSON:
		f = &JSONFormatter{}

	default:
		panic(fmt.Sprintf("unknown style %q", style))
	}

	return &Writer{
		w: w,
		f: f,
	}
}

// Write implements the io.Writer interface for a log writer.
func (w *Writer) Write(b []byte) (int, error) {
	var newline string
	if bytes.HasSuffix(b, []byte("\n")) {
		newline = "\n"
	}

	pc, file, line, _ := runtime.Caller(3)
	if strings.Contains(file, "/pkg/logger/") {
		pc, file, line, _ = runtime.Caller(4)
	}

	at := time.Now()
	funcName := runtime.FuncForPC(pc).Name()

	message := string(b)
	message = w.f.Format(message, newline, at, funcName, file, line)
	message = strings.ReplaceAll(message, info.Main.Path+"/", "")
	message = strings.ReplaceAll(message, exedir+"/", "")

	w.mu.Lock()
	defer w.mu.Unlock()

	return w.w.Write([]byte(message))
}
