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

	"github.com/polyscone/tofu/internal/pkg/errsx"
)

const (
	SkipFile SkipRuleKind = iota
	SkipFunc
)

var (
	exedir    = filepath.ToSlash(filepath.Dir(errsx.Must(os.Executable())))
	info      = errsx.MustOK(debug.ReadBuildInfo())
	skipRules = []skipRule{}
)

func init() {
	AddSkipRule("/pkg/logger/", SkipFile)
}

type SkipRuleKind byte

type skipRule struct {
	value string
	kind  SkipRuleKind
}

func AddSkipRule(value string, kind SkipRuleKind) {
	skipRules = append(skipRules, skipRule{
		value: value,
		kind:  kind,
	})
}

// Formatter represents a formatter that can be used by a
// logger to format messages.
type Formatter interface {
	Format(message, newline string, at time.Time) string
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

	at := time.Now()
	skip := 3
	pc, file, _, _ := runtime.Caller(skip)
	funcName := runtime.FuncForPC(pc).Name()

skipLoop:
	for _, rule := range skipRules {
		switch {
		case rule.kind == SkipFile && strings.Contains(file, rule.value),
			rule.kind == SkipFunc && strings.Contains(funcName, rule.value):

			skip++
			pc, file, _, _ = runtime.Caller(skip)
			funcName = runtime.FuncForPC(pc).Name()

			goto skipLoop
		}
	}

	message := string(b)
	message = w.f.Format(message, newline, at)
	message = strings.ReplaceAll(message, info.Main.Path+"/", "")
	message = strings.ReplaceAll(message, exedir+"/", "")

	w.mu.Lock()
	defer w.mu.Unlock()

	return w.w.Write([]byte(message))
}
