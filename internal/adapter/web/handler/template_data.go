package handler

import (
	"context"
	"fmt"
	"html/template"
	"net/url"

	"github.com/polyscone/tofu/internal/adapter/web/httputil"
)

type CSRF struct {
	Ctx context.Context
}

func (c CSRF) Token() string {
	return httputil.MaskedCSRFToken(c.Ctx)
}

type Form struct {
	url.Values
}

func (f Form) GetOr(key string, fallback any) string {
	if f.Values == nil {
		return fmt.Sprintf("%v", fallback)
	}

	return f.Get(key)
}

func (f Form) GetAll(key string) []string {
	return f.Values[key]
}

func (f Form) GetAllOr(key string, fallback any) ([]string, error) {
	if f.Values == nil {
		return TmplToStrings(fallback)
	}

	return f.Values[key], nil
}

type Query struct {
	url.Values
}

func (q Query) String(pairs ...any) (template.URL, error) {
	return TmplQueryString(q.Values, pairs...)
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
	ShortName   string
	Description string
	ThemeColour string
}

type SessionData struct {
	// General session keys
	Flash          []string
	FlashImportant []string
	Redirect       string
	HighlightID    int

	// Account session keys
	UserID                   int
	Email                    string
	TOTPMethod               string
	HasActivatedTOTP         bool
	IsAwaitingTOTP           bool
	IsSignedIn               bool
	KnownPasswordBreachCount int
}