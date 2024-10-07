package handler

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"path"
	"strings"

	"github.com/polyscone/tofu/app"
	"github.com/polyscone/tofu/httpx"
)

type AssetPipeline struct {
	scope string
	rn    *Renderer
	r     *http.Request
}

func (a AssetPipeline) Tag(location string) string {
	original := app.BasePath + location
	tagged, ok := a.rn.AssetLocationTag(original)
	if !a.rn.h.Tenant.Dev && ok {
		return tagged
	}

	_, _, b, err := a.rn.Asset(a.r, location)
	if err != nil {
		return original
	}

	hash := md5.New()
	if _, err := hash.Write(b); err != nil {
		return original
	}

	tag := hex.EncodeToString(hash.Sum(nil))
	ext := path.Ext(location)
	tagged = strings.TrimSuffix(location, ext) + "." + tag + ext

	a.rn.TagAsset(original, tagged)

	return tagged
}

type CSRF struct {
	Ctx context.Context
}

func (c CSRF) Token() string {
	return httpx.MaskedCSRFToken(c.Ctx)
}

type Form struct {
	url.Values
}

func (f Form) GetOr(key string, fallback any) string {
	if _, ok := f.Values[key]; !ok {
		return fmt.Sprintf("%v", fallback)
	}

	return f.Get(key)
}

func (f Form) GetAll(key string) []string {
	return f.Values[key]
}

func (f Form) GetAllOr(key string, fallback any) ([]string, error) {
	if _, ok := f.Values[key]; !ok {
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
	if _, ok := q.Values[key]; !ok {
		return fmt.Sprintf("%v", fallback)
	}

	return q.Get(key)
}

func (q Query) GetAll(key string) []string {
	return q.Values[key]
}

func (q Query) GetAllOr(key string, fallback any) ([]string, error) {
	if _, ok := q.Values[key]; !ok {
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
	BasePath    string
}

type SessionData struct {
	// General session keys
	Flash          []string
	FlashWarning   []string
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
