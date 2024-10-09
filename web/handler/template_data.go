package handler

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"path"
	"slices"
	"strings"

	"github.com/polyscone/tofu/app"
	"github.com/polyscone/tofu/httpx"
)

type AssetPipeline struct {
	scope   string
	rn      *Renderer
	r       *http.Request
	imports []string
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

func (a *AssetPipeline) tag(asset string) (string, string, string) {
	key := asset
	tagged := asset

	u, err := url.Parse(asset)
	if err != nil {
		return key, asset, tagged
	}

	ctx := context.Background()
	r := a.r.Clone(ctx)

	r.URL.Path = u.Path
	r.URL.RawQuery = u.RawQuery

	_, _, b, err := a.rn.Asset(r, a, u.Path)
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
	asset = u.Path

	return key, asset, tagged
}

func (a *AssetPipeline) Tag(asset string) string {
	asset = a.resolve(asset)

	key, asset, tagged := a.tag(asset)

	a.rn.TagAsset(key, asset, tagged)

	return tagged
}

func (a *AssetPipeline) TagImport(asset string) string {
	asset = a.resolve(asset)

	a.imports = append(a.imports, asset)

	key, asset, tagged := a.tag(asset)

	a.rn.TagAsset(key, asset, tagged)

	return asset
}

func (a *AssetPipeline) ImportMap() string {
	slices.Sort(a.imports)

	a.imports = slices.Compact(a.imports)
	if len(a.imports) == 0 {
		return ""
	}

	imports := make(map[string]string, len(a.imports))
	for _, im := range a.imports {
		tagged, ok := a.rn.FindTaggedByAsset(im)
		if !ok {
			continue
		}

		imports[im] = tagged
	}

	b, err := json.Marshal(map[string]any{"imports": imports})
	if err != nil {
		return ""
	}

	return string(b)
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
