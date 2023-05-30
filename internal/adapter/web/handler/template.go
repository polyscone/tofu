package handler

import (
	"context"
	"encoding/base64"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/polyscone/tofu/internal/pkg/csrf"
	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/http/router"
)

type CSRF struct {
	ctx context.Context
}

func (c CSRF) Token() string {
	return base64.RawURLEncoding.EncodeToString(csrf.MaskedToken(c.ctx))
}

type Query struct {
	url.Values
}

func (q Query) String(pairs ...any) (template.URL, error) {
	return tmplQueryString(q.Values, pairs...)
}

type Vars map[string]any

func (v Vars) Merge(rhs Vars) Vars {
	if rhs == nil {
		return v
	}

	if v == nil {
		v = make(Vars, len(rhs))
	}

	for key, value := range rhs {
		v[key] = value
	}

	return v
}

type URL struct {
	Scheme   string
	Host     string
	Hostname string
	Port     string
	Path     template.URL
	Query    Query
}

type AppData struct {
	Name        string
	Description string
	HasSMS      bool
}

type SessionData struct {
	// General session keys
	Flash          template.HTML
	FlashImportant bool
	Redirect       string

	// Account session keys
	UserID                   int
	Email                    string
	TOTPMethod               string
	HasActivatedTOTP         bool
	IsAwaitingTOTP           bool
	IsAuthenticated          bool
	PasswordKnownBreachCount int
}

type ViewData struct {
	View         string
	Status       int
	CSRF         CSRF
	ErrorMessage string
	Errors       errors.Map
	Form         url.Values
	URL          URL
	App          AppData
	Session      SessionData
	Com          any
	Vars         Vars
}

func (v ViewData) ComData(data any) ViewData {
	v.Com = data

	return v
}

type ViewDataFunc func(data *ViewData)

type emailData struct {
	URL  URL
	App  AppData
	Vars Vars
}

type emailDataFunc func(data *emailData)

func tmplAdd(a, b int) int {
	return a + b
}

func tmplSub(a, b int) int {
	return a - b
}

func tmplMul(a, b int) int {
	return a * b
}

func tmplDiv(a, b int) int {
	return a / b
}

func tmplMod(a, b int) int {
	return a % b
}

func tmplInts(start, end int) []int {
	n := end - start
	ints := make([]int, n)
	for i := 0; i < n; i++ {
		ints[i] = start + i
	}

	return ints
}

func tmplStatusText(code int) string {
	return http.StatusText(code)
}

type tmplPathFunc func(name string, paramArgPairs ...any) template.URL

func tmplPath(mux *router.ServeMux) tmplPathFunc {
	return func(name string, paramArgPairs ...any) template.URL {
		return template.URL(mux.Path(name, paramArgPairs...))
	}
}

func tmplQueryReplace(q url.Values, pairs ...any) (url.Values, error) {
	if len(pairs)%2 == 1 {
		return nil, errors.Tracef("QueryString expects pairs of key value replacements")
	}

	u, err := url.Parse("?" + q.Encode())
	if err != nil {
		return nil, errors.Tracef(err)
	}

	q = u.Query()
	for i := 0; i < len(pairs); i += 2 {
		key := fmt.Sprintf("%v", pairs[i])
		value := pairs[i+1]

		if value == nil {
			q.Del(key)

			continue
		}

		q.Set(key, fmt.Sprintf("%v", value))
	}

	return q, nil
}

func tmplQueryURL(q url.Values) template.URL {
	value := q.Encode()

	if value == "" {
		return ""
	}

	if !strings.HasPrefix(value, "?") {
		value = "?" + value
	}

	return template.URL(value)
}

func tmplQueryString(q url.Values, pairs ...any) (template.URL, error) {
	q, err := tmplQueryReplace(q, pairs...)
	if err != nil {
		return "", errors.Tracef(err)
	}

	for key, values := range q {
		var keep bool
		for _, value := range values {
			if value != "" {
				keep = true

				break
			}
		}

		if !keep {
			q.Del(key)
		}
	}

	return tmplQueryURL(q), nil
}

func tmplFormatTime(t time.Time, format string) string {
	switch format {
	case "DateTime":
		return t.Format(time.DateTime)

	case "RFC3339":
		return t.Format(time.RFC3339)
	}

	return t.Format(format)
}
