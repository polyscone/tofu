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

	values := f.Values[key]
	if values == nil {
		return TmplToStrings(fallback)
	}

	return values, nil
}

type Query struct {
	url.Values
}

func (q Query) GetOr(key string, fallback any) string {
	if q.Values == nil {
		return fmt.Sprintf("%v", fallback)
	}

	return q.Get(key)
}

func (q Query) GetAll(key string) []string {
	return q.Values[key]
}

func (q Query) GetAllOr(key string, fallback any) ([]string, error) {
	if q.Values == nil {
		return TmplToStrings(fallback)
	}

	values := q.Values[key]
	if values == nil {
		return TmplToStrings(fallback)
	}

	return values, nil
}

func (q Query) String() template.URL {
	return TmplQueryString(q.Values)
}

func (q Query) Replace(pairs ...any) (template.URL, error) {
	return TmplQueryReplace(q.Values, pairs...)
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
	Scheme string
	Host   string
	Path   template.URL
	Query  Query
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
	FlashError     []string
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
