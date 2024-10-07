package handler

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"mime"
	"net/http"
	"path"
	"strings"
	texttemplate "text/template"
	"time"

	"github.com/polyscone/tofu/app"
	"github.com/polyscone/tofu/app/account"
	"github.com/polyscone/tofu/app/system"
	"github.com/polyscone/tofu/cache"
	"github.com/polyscone/tofu/errsx"
	"github.com/polyscone/tofu/httpx"
	"github.com/polyscone/tofu/web/guard"
)

var ErrNoIndex = errors.New("no index file")

type State struct {
	data map[string]any
}

func (s *State) Get(key string) any {
	return s.data[key]
}

func (s *State) Set(key string, value any) {
	if s.data == nil {
		s.data = make(map[string]any)
	}

	s.data[key] = value
}

func (s *State) Store(key string, value any) bool {
	if s.data == nil {
		s.data = make(map[string]any)
	}

	if _, ok := s.data[key]; ok {
		return false
	}

	s.data[key] = value

	return true
}

type ViewData struct {
	Asset        AssetPipeline
	View         string
	Stream       string
	Status       int
	CSRF         CSRF
	ErrorMessage string
	Errors       errsx.Map
	Now          time.Time
	Form         Form
	URL          URL
	App          AppData
	Session      SessionData
	Config       *system.Config
	User         *account.User
	Passport     guard.Passport
	Props        map[string]any
	State        *State
	Vars         Vars
}

func (v ViewData) WithProps(pairs ...any) (ViewData, error) {
	if len(pairs)%2 == 1 {
		return v, errors.New("WithProps: want key value pairs")
	}

	v.Props = make(map[string]any, len(pairs)/2)
	for i := 0; i < len(pairs); i += 2 {
		key := fmt.Sprintf("%v", pairs[i])
		value := pairs[i+1]

		if key == "Props" {
			props, ok := value.(map[string]any)
			if ok {
				for key, value := range props {
					v.Props[key] = value
				}

				continue
			}
		}

		v.Props[key] = value
	}

	return v, nil
}

type Templater interface {
	Execute(wr io.Writer, data any) error
}

type ViewDataFunc func(data *ViewData) error
type ViewVarsFunc func(r *http.Request) (Vars, error)
type TemplatePathsFunc func(view string) []string
type TemplateProcessFunc func(w http.ResponseWriter, r *http.Request)

type RendererConfig struct {
	Handler           *Handler
	AssetTagLocations *cache.Cache[string, string]
	AssetFiles        fs.FS
	TemplateFiles     fs.FS
	TemplatePaths     TemplatePathsFunc
	Funcs             template.FuncMap
	Process           TemplateProcessFunc
}

type Renderer struct {
	h                 *Handler
	assetTagLocations *cache.Cache[string, string]
	assetFiles        fs.FS
	templateFiles     fs.FS
	templatePaths     TemplatePathsFunc
	funcs             template.FuncMap
	viewVarsFuncs     map[string]ViewVarsFunc
	process           TemplateProcessFunc
}

func NewRenderer(config RendererConfig) *Renderer {
	return &Renderer{
		h:                 config.Handler,
		assetTagLocations: config.AssetTagLocations,
		assetFiles:        config.AssetFiles,
		templateFiles:     config.TemplateFiles,
		templatePaths:     config.TemplatePaths,
		funcs:             config.Funcs,
		viewVarsFuncs:     make(map[string]ViewVarsFunc),
		process:           config.Process,
	}
}

func (rn *Renderer) data(ctx context.Context, r *http.Request, status int, view string) (ViewData, error) {
	config := rn.h.Config(ctx)
	user := rn.h.User(ctx)
	passport := rn.h.Passport(ctx)

	data := ViewData{
		Asset: AssetPipeline{
			rn: rn,
			r:  r,
		},
		View:   view,
		Status: status,
		CSRF:   CSRF{Ctx: ctx},
		Now:    time.Now(),
		Form:   Form{Values: r.PostForm},
		URL: URL{
			Scheme: rn.h.Tenant.Scheme,
			Host:   rn.h.Tenant.Host,
			Path:   template.URL(r.URL.Path),
			Query:  Query{Values: r.URL.Query()},
		},
		App: AppData{
			Name:        app.Name,
			ShortName:   app.ShortName,
			Description: app.Description,
			ThemeColour: app.ThemeColour,
			BasePath:    app.BasePath,
		},
		Session: SessionData{
			// Global session keys
			Flash:          rn.h.Session.PopFlash(ctx),
			FlashWarning:   rn.h.Session.PopFlashWarning(ctx),
			FlashImportant: rn.h.Session.PopFlashImportant(ctx),
			FlashError:     rn.h.Session.PopFlashError(ctx),
			Redirect:       rn.h.Session.Redirect(ctx),
			HighlightID:    rn.h.Session.PopHighlightID(ctx),

			// Account session keys
			UserID:                   rn.h.Session.UserID(ctx),
			Email:                    rn.h.Session.Email(ctx),
			TOTPMethod:               rn.h.Session.TOTPMethod(ctx),
			HasActivatedTOTP:         rn.h.Session.HasActivatedTOTP(ctx),
			IsAwaitingTOTP:           rn.h.Session.IsAwaitingTOTP(ctx),
			IsSignedIn:               rn.h.Session.IsSignedIn(ctx),
			KnownPasswordBreachCount: rn.h.Session.KnownPasswordBreachCount(ctx),
		},
		Config:   config,
		User:     user,
		Passport: passport,
		State:    &State{},
	}

	if vars, ok := rn.viewVarsFuncs[view]; ok {
		defaults, err := vars(r)
		if err != nil {
			return data, fmt.Errorf("vars: %w", err)
		}

		data.Vars = data.Vars.Merge(defaults)
	}

	// Make sure the current view name isn't overwritten by a user function
	data.View = view

	return data, nil
}

func (rn *Renderer) ViewFunc(w http.ResponseWriter, r *http.Request, status int, view string, dataFunc ViewDataFunc) {
	ctx := r.Context()
	tmpl := rn.h.Template(rn.templateFiles, rn.templatePaths(view), rn.funcs, view)

	data, err := rn.data(ctx, r, status, view)
	if err != nil {
		rn.h.Logger(ctx).Error("view data", "error", err)

		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)

		return
	}

	if dataFunc != nil {
		if err := dataFunc(&data); err != nil {
			rn.h.Logger(ctx).Error("execute view data func", "error", err)

			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)

			return
		}
	}

	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, "master", data); err != nil {
		rn.h.Logger(ctx).Error("execute view template", "error", err)

		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)

		return
	}

	if rn.process != nil {
		rn.process(w, r)
	}

	if status != 0 {
		w.WriteHeader(status)
	}

	if _, err := buf.WriteTo(w); err != nil {
		rn.h.Logger(ctx).Error("write view template response", "error", err)
	}
}

func (rn *Renderer) View(w http.ResponseWriter, r *http.Request, status int, view string, vars Vars) {
	rn.ViewFunc(w, r, status, view, func(data *ViewData) error {
		data.Vars = data.Vars.Merge(vars)

		return nil
	})
}

func (rn *Renderer) StreamView(w http.ResponseWriter, r *http.Request, status int, view string, vars Vars) func() {
	rn.ViewFunc(w, r, status, view, func(data *ViewData) error {
		data.Stream = "begin"
		data.Vars = data.Vars.Merge(vars)

		return nil
	})

	return func() {
		rn.ViewFunc(w, r, 0, view, func(data *ViewData) error {
			data.Stream = "end"
			data.Vars = data.Vars.Merge(vars)

			return nil
		})
	}
}

func (rn *Renderer) SetViewVars(name string, vars ViewVarsFunc) {
	if _, ok := rn.viewVarsFuncs[name]; ok {
		panic(fmt.Sprintf("default view vars already set for %q", name))
	}

	rn.viewVarsFuncs[name] = vars
}

func (rn *Renderer) Text(dst io.Writer, r *http.Request, status int, text string, vars Vars) error {
	ctx := r.Context()
	data, err := rn.data(ctx, r, status, "text_template")
	if err != nil {
		return fmt.Errorf("text template data: %w", err)
	}

	data.Vars = data.Vars.Merge(vars)

	tmpl := texttemplate.New("").Option("missingkey=default").Funcs(rn.funcs)
	if _, err := tmpl.Parse(text); err != nil {
		return fmt.Errorf("parse text template: %w", err)
	}

	if err := tmpl.Execute(dst, data); err != nil {
		return fmt.Errorf("execute text template: %w", err)
	}

	return nil
}

func (rn *Renderer) HTML(dst io.Writer, r *http.Request, status int, html string, vars Vars) error {
	ctx := r.Context()
	data, err := rn.data(ctx, r, status, "html_template")
	if err != nil {
		return fmt.Errorf("HTML template data: %w", err)
	}

	data.Vars = data.Vars.Merge(vars)

	tmpl := template.New("").Option("missingkey=default").Funcs(rn.funcs)
	if _, err := tmpl.Parse(html); err != nil {
		return fmt.Errorf("parse HTML template: %w", err)
	}

	if err := tmpl.Execute(dst, data); err != nil {
		return fmt.Errorf("execute HTML template: %w", err)
	}

	return nil
}

func (rn *Renderer) HandlerFunc(view string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rn.View(w, r, http.StatusOK, view, nil)
	}
}

func (rn *Renderer) ErrorViewFunc(w http.ResponseWriter, r *http.Request, msg string, err error, view string, dataFunc ViewDataFunc) {
	ctx := r.Context()

	rn.h.Logger(ctx).Error(msg, "error", err)

	status := httpx.ErrorStatus(err)

	if status == http.StatusTooManyRequests {
		// If a client is hitting a rate limit we set the connection header to
		// close which will trigger the standard library's HTTP server to close
		// the connection after the response is sent
		//
		// Doing this means the client needs to go through the handshake process
		// again to make a new connection the next time, which should help to slow
		// down additional requests for clients that keep on hitting the limit
		w.Header().Set("connection", "close")
	}

	rn.ViewFunc(w, r, status, view, func(data *ViewData) error {
		data.ErrorMessage = httpx.ErrorMessage(err)

		switch {
		case errors.Is(err, app.ErrMalformedInput),
			errors.Is(err, app.ErrInvalidInput),
			errors.Is(err, app.ErrConflict):

			var errs errsx.Map
			if errors.As(err, &errs) {
				data.Errors = errs
			}
		}

		if dataFunc != nil {
			return dataFunc(data)
		}

		return nil
	})
}

func (rn *Renderer) ErrorView(w http.ResponseWriter, r *http.Request, msg string, err error, view string, vars Vars) {
	rn.ErrorViewFunc(w, r, msg, err, view, func(data *ViewData) error {
		data.Vars = data.Vars.Merge(vars)

		return nil
	})
}

func (rn *Renderer) Asset(r *http.Request, upath string) (string, time.Time, []byte, error) {
	fpath := strings.TrimPrefix(upath, "/")
	f, err := rn.assetFiles.Open(fpath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) || errors.Is(err, fs.ErrInvalid) {
			return "", time.Time{}, nil, fmt.Errorf("%w: %w", httpx.ErrNotFound, err)
		}

		return "", time.Time{}, nil, fmt.Errorf("%w: %w", httpx.ErrInternalServerError, err)
	}
	defer f.Close()

	stat, err := f.Stat()
	if err != nil {
		return "", time.Time{}, nil, fmt.Errorf("%w: stat: %w", httpx.ErrInternalServerError, err)
	}

	// If the directory has either an index.html or index.htm file then display that
	// otherwise we forbid viewing directories
	if stat.IsDir() {
		if !strings.HasSuffix(fpath, "/") {
			fpath += "/"
		}
		if !strings.HasSuffix(upath, "/") {
			upath += "/"
		}

		var hasIndex bool
		for _, name := range []string{"index.html", "index.htm"} {
			if _f, err := rn.assetFiles.Open(fpath + name); err == nil {
				defer _f.Close()

				_stat, err := _f.Stat()
				if err != nil {
					return "", time.Time{}, nil, fmt.Errorf("%w: stat: %w", httpx.ErrInternalServerError, err)
				}

				f = _f
				stat = _stat
				fpath += name
				upath += name
				hasIndex = true

				break
			}
		}

		if !hasIndex {
			return "", time.Time{}, nil, fmt.Errorf("%w: directory: %w", httpx.ErrForbidden, ErrNoIndex)
		}
	}

	b, err := io.ReadAll(f)
	if err != nil {
		return "", time.Time{}, nil, fmt.Errorf("%w: read all: %w", httpx.ErrInternalServerError, err)
	}

	modtime := stat.ModTime()
	if bytes.Contains(b, []byte("{{")) && bytes.Contains(b, []byte("}}")) {
		modtime = time.Time{}

		contentType := mime.TypeByExtension(path.Ext(upath))
		if contentType == "" {
			contentType = http.DetectContentType(b)
		}
		mediaType, _, _ := mime.ParseMediaType(contentType)
		if mediaType == "" {
			mediaType = "application/octet-stream"
		}

		render := rn.Text
		if mediaType == "text/html" {
			render = rn.HTML
		}

		var buf bytes.Buffer
		if err := render(&buf, r, http.StatusOK, string(b), nil); err != nil {
			return "", time.Time{}, nil, fmt.Errorf("%w: render: %w", httpx.ErrInternalServerError, err)
		}

		b = buf.Bytes()
	}

	return stat.Name(), modtime, b, nil
}

func (rn *Renderer) TagAsset(location, tag string) {
	if rn.assetTagLocations == nil {
		return
	}

	rn.assetTagLocations.LoadOrStore(tag, func() string { return location })
	rn.assetTagLocations.LoadOrStore("loc:"+location, func() string { return tag })
}

func (rn *Renderer) AssetTagLocation(tag string) (string, bool) {
	if rn.assetTagLocations == nil {
		return "", false
	}

	return rn.assetTagLocations.Load(tag)
}

func (rn *Renderer) AssetLocationTag(location string) (string, bool) {
	if rn.assetTagLocations == nil {
		return "", false
	}

	return rn.assetTagLocations.Load("loc:" + location)
}
