package handler

import (
	"context"
	"encoding/base64"
	"html/template"
	"net/url"

	"github.com/polyscone/tofu/internal/adapter/web/passport"
	"github.com/polyscone/tofu/internal/pkg/csrf"
	"github.com/polyscone/tofu/internal/pkg/errors"
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
	Flash          []string
	FlashImportant []string
	Redirect       string

	// Account session keys
	UserID                   int
	Email                    string
	TOTPMethod               string
	HasActivatedTOTP         bool
	IsAwaitingTOTP           bool
	IsSignedIn               bool
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
	Guard        passport.Passport
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
