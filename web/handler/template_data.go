package handler

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"net/url"
	"path"
	"slices"
	"strings"

	"github.com/polyscone/tofu/app"
	"github.com/polyscone/tofu/internal/httpx"
)

type AssetPipeline struct {
	scope            string
	rn               *Renderer
	r                *http.Request
	cssLinks         []string
	htmlTemplates    []string
	jsImportMappings []string
	jsImports        []string
}

func (a *AssetPipeline) resolve(asset string) string {
	isRelative := strings.HasPrefix(asset, "./") || strings.HasPrefix(asset, "../")
	if isRelative {
		asset = path.Join(path.Dir(a.r.URL.Path), asset)
	}

	if !strings.HasPrefix(asset, app.BasePath) {
		asset = app.BasePath + asset
	}

	return asset
}

func (a *AssetPipeline) data(asset string) ([]byte, string, error) {
	u, err := url.Parse(asset)
	if err != nil {
		return nil, "", fmt.Errorf("URL parse: %w", err)
	}

	ctx := context.Background()
	r := a.r.Clone(ctx)

	r.URL.Path = u.Path
	r.URL.RawQuery = u.RawQuery

	_, _, b, err := a.rn.Asset(r, a, asset)
	if err != nil {
		return b, u.Path, fmt.Errorf("renderer asset: %w", err)
	}

	return b, u.Path, nil
}

func (a *AssetPipeline) tag(asset string) (string, string, string) {
	key := asset
	tagged := asset

	b, upath, err := a.data(asset)
	if err != nil {
		return key, asset, tagged
	}

	hash := md5.New()
	if _, err := hash.Write(b); err != nil {
		return key, asset, tagged
	}

	hashsum := hex.EncodeToString(hash.Sum(nil))
	ext := path.Ext(asset)
	tagged = strings.TrimSuffix(asset, ext) + "." + hashsum + ext

	// We only set the asset to the path without query string here because
	// we want to make sure the tagged itself preserved the query string
	asset = upath

	return key, asset, tagged
}

func (a *AssetPipeline) Tag(asset string) string {
	asset = a.resolve(asset)

	key, asset, tagged := a.tag(asset)

	if asset != tagged {
		a.rn.TagAsset(key, asset, tagged)
	}

	return tagged
}

func (a *AssetPipeline) TagJSImport(asset string) string {
	asset = a.resolve(asset)

	a.jsImportMappings = append(a.jsImportMappings, asset)

	key, asset, tagged := a.tag(asset)

	if asset != tagged {
		a.rn.TagAsset(key, asset, tagged)
	}

	return asset
}

func (a *AssetPipeline) comment(message string) any {
	switch path.Ext(a.r.URL.Path) {
	case ".css", ".js":
		return template.JS(fmt.Sprintf("/* %v */", message))

	default:
		return template.HTML(fmt.Sprintf("<!-- %v -->", message))
	}
}

func (a *AssetPipeline) Register(asset string, args ...string) any {
	switch path.Ext(asset) {
	case ".css":
		asset = a.Tag(asset)

		a.cssLinks = append(a.cssLinks, asset)

	case ".html":
		if len(args) == 0 {
			return a.comment("an additional template id argument is required when registering .html files")
		}

		id := args[0]
		asset = a.resolve(asset)

		b, _, err := a.data(asset)
		if err != nil {
			break
		}

		tmpl := `<template id="` + id + `">` + string(b) + `</template>`

		a.htmlTemplates = append(a.htmlTemplates, tmpl)

	case ".js":
		asset = a.TagJSImport(asset)

		a.jsImports = append(a.jsImports, asset)

	default:
		message := fmt.Sprintf("unsupported register file extension in %q", asset)

		return a.comment(message)
	}

	return ""
}

func (a *AssetPipeline) CSSLinks() string {
	slices.Sort(a.cssLinks)

	a.cssLinks = slices.Compact(a.cssLinks)
	if len(a.cssLinks) == 0 {
		return ""
	}

	links := make([]string, len(a.cssLinks))
	for i, link := range a.cssLinks {
		links[i] = fmt.Sprintf(`<link rel="stylesheet" href=%q>`, link)
	}

	return strings.Join(links, "\n")
}

func (a *AssetPipeline) HTMLTemplates() string {
	slices.Sort(a.htmlTemplates)

	a.htmlTemplates = slices.Compact(a.htmlTemplates)
	if len(a.htmlTemplates) == 0 {
		return ""
	}

	return strings.Join(a.htmlTemplates, "\n")
}

func (a *AssetPipeline) JSImportMap() string {
	slices.Sort(a.jsImportMappings)

	a.jsImportMappings = slices.Compact(a.jsImportMappings)
	if len(a.jsImportMappings) == 0 {
		return ""
	}

	imports := make(map[string]string, len(a.jsImportMappings))
	for _, im := range a.jsImportMappings {
		tagged, ok := a.rn.FindTaggedByAsset(im)
		if !ok {
			continue
		}

		imports[im] = tagged
	}

	b, err := json.MarshalIndent(map[string]any{"imports": imports}, "", "\t")
	if err != nil {
		return ""
	}

	return string(b)
}

func (a *AssetPipeline) JSImports() string {
	slices.Sort(a.jsImports)

	a.jsImports = slices.Compact(a.jsImports)
	if len(a.jsImports) == 0 {
		return ""
	}

	imports := make([]string, len(a.jsImports))
	for i, module := range a.jsImports {
		imports[i] = fmt.Sprintf("import %q", module)
	}

	return strings.Join(imports, "; ")
}

func (a *AssetPipeline) WriteCSSLinks() template.HTML {
	return "<!-- Renderer: CSS links -->"
}

func (a *AssetPipeline) WriteHTMLTemplates() template.HTML {
	return "<!-- Renderer: HTML templates -->"
}

func (a *AssetPipeline) WriteJSImportMap() template.HTML {
	return "<!-- Renderer: JS import map -->"
}

func (a *AssetPipeline) WriteJSImports() template.HTML {
	return "<!-- Renderer: JS imports -->"
}

type Logger struct {
	logger *slog.Logger
}

func (l Logger) Info(msg string, args ...any) string {
	l.logger.Info(msg, args...)

	return ""
}

func (l Logger) Warn(msg string, args ...any) string {
	l.logger.Warn(msg, args...)

	return ""
}

func (l Logger) Error(msg string, args ...any) string {
	l.logger.Error(msg, args...)

	return ""
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
