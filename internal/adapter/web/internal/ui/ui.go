package ui

import (
	"bytes"
	"embed"
	"encoding/base64"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"text/template"

	"github.com/polyscone/tofu/internal/adapter/web/internal/httputil"
	"github.com/polyscone/tofu/internal/adapter/web/internal/sesskey"
	"github.com/polyscone/tofu/internal/adapter/web/internal/smtp"
	"github.com/polyscone/tofu/internal/adapter/web/internal/token"
	"github.com/polyscone/tofu/internal/pkg/command"
	"github.com/polyscone/tofu/internal/pkg/csrf"
	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/fstack"
	"github.com/polyscone/tofu/internal/pkg/http/router"
	"github.com/polyscone/tofu/internal/pkg/session"
)

//go:embed "files/static" "files/template"
var embeddedFiles embed.FS

var tmplFuncs = template.FuncMap{
	"StatusText": http.StatusText,
}

type Option func(app *App)

func WithDev(value bool) Option {
	return func(app *App) {
		app.dev = value
	}
}

type App struct {
	dev         bool
	bus         command.Bus
	sessions    *session.Manager
	tokens      token.Repo
	mailer      smtp.Mailer
	files       fs.FS
	templatesMu sync.RWMutex
	templates   map[string]*template.Template
}

func New(bus command.Bus, sessions *session.Manager, tokens token.Repo, mailer smtp.Mailer, opts ...Option) *App {
	files := fs.FS(embeddedFiles)
	templates := make(map[string]*template.Template)

	app := App{
		bus:       bus,
		sessions:  sessions,
		tokens:    tokens,
		mailer:    mailer,
		files:     files,
		templates: templates,
	}

	for _, opt := range opts {
		opt(&app)
	}

	if app.dev {
		dir := "internal/adapter/web/internal/ui"
		if info, err := os.Stat(dir); err == nil && info.IsDir() {
			app.files = fstack.New(os.DirFS(dir), app.files)
		}
	}

	return &app
}

func (app *App) Routes() http.Handler {
	static := errors.Must(fs.Sub(app.files, "files/static"))

	mux := router.NewServeMux()

	mux.Redirect(http.MethodGet, "/favicon.ico", "/favicon.png", http.StatusTemporaryRedirect)

	mux.Get("/", app.homeGet)

	mux.Prefix("/account", func(mux *router.ServeMux) {
		mux.Get("/activate", app.accountActivateGet)
		mux.Post("/activate", app.accountActivatePost)

		mux.Get("/register", app.accountRegisterGet)
		mux.Post("/register", app.accountRegisterPost)

		mux.Get("/login", app.accountLoginGet)
		mux.Post("/login", app.accountLoginPost)

		mux.Post("/logout", app.accountLogoutPost)

		mux.Get("/forgotten-password", app.accountForgottenPasswordGet)
	})

	mux.GetHandler("/:rest", http.FileServer(http.FS(static)))

	mux.NotFound(func(w http.ResponseWriter, r *http.Request) {
		app.renderError(w, r, errors.Tracef("%w: %v %v", httputil.ErrNotFound, r.Method, r.URL))
	})

	mux.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
		app.renderError(w, r, errors.Tracef("%w: %v %v", httputil.ErrMethodNotAllowed, r.Method, r.URL))
	})

	return mux
}

func (app *App) ErrorHandler(w http.ResponseWriter, r *http.Request, err error) {
	app.renderError(w, r, errors.Tracef(err))
}

func (app *App) csrfToken(r *http.Request) string {
	ctx := r.Context()

	return base64.RawURLEncoding.EncodeToString(csrf.MaskedToken(ctx))
}

func (app *App) view(view string) *template.Template {
	app.templatesMu.RLock()

	// Return the cached template only when we're not in a dev environment
	if tmpl := app.templates[view]; tmpl != nil && !app.dev {
		app.templatesMu.RUnlock()

		return tmpl
	}

	app.templatesMu.RUnlock()

	app.templatesMu.Lock()
	defer app.templatesMu.Unlock()

	key := strings.TrimSuffix(filepath.Base(view), ".go.html")

	tmpl := template.New(key).Option("missingkey=zero").Funcs(tmplFuncs)
	tmpl = errors.Must(tmpl.ParseFS(app.files, "files/template/master.go.html"))
	tmpl = errors.Must(tmpl.ParseFS(app.files, "files/template/partial/*.go.html"))
	tmpl = errors.Must(tmpl.ParseFS(app.files, "files/template/view/"+view+".go.html"))

	app.templates[key] = tmpl

	return tmpl
}

type sessionRenderData struct {
	UserID         string
	Email          string
	IsAwaitingTOTP bool
}

type registerRenderData struct {
	Email string
}

type renderData struct {
	// Generic render data
	Status       int
	CSRFToken    string
	ErrorMessage string
	Errors       errors.Map
	PostForm     map[string]string
	Query        map[string]string
	Session      sessionRenderData

	// View-specific render data
	Register registerRenderData
}

type renderDataFunc func(data *renderData)

func (app *App) render(w http.ResponseWriter, r *http.Request, status int, view string, dataFunc renderDataFunc) {
	var buf bytes.Buffer
	var postForm map[string]string
	var query map[string]string

	ctx := r.Context()

	if r.PostForm != nil {
		postForm = make(map[string]string, len(r.PostForm))

		for key, values := range r.PostForm {
			postForm[key] = values[0]
		}
	}

	if q := r.URL.Query(); q != nil {
		query = make(map[string]string, len(q))

		for key, values := range q {
			query[key] = values[0]
		}
	}

	data := renderData{
		CSRFToken: app.csrfToken(r),
		Status:    status,
		PostForm:  postForm,
		Query:     query,
		Session: sessionRenderData{
			UserID:         app.sessions.GetString(ctx, sesskey.UserID),
			Email:          app.sessions.GetString(ctx, sesskey.Email),
			IsAwaitingTOTP: app.sessions.GetBool(ctx, sesskey.IsAwaitingTOTP),
		},
	}

	if dataFunc != nil {
		dataFunc(&data)
	}

	if err := app.view(view).ExecuteTemplate(&buf, "master", data); err != nil {
		httputil.LogError(r, errors.Tracef(err))

		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)

		return
	}

	w.Header().Set("content-type", "text/html")
	w.WriteHeader(status)

	if _, err := buf.WriteTo(w); err != nil {
		httputil.LogError(r, errors.Tracef(err))
	}
}

func (app *App) renderErrorView(w http.ResponseWriter, r *http.Request, err error, view string, dataFunc renderDataFunc) bool {
	if err == nil {
		return false
	}

	httputil.LogError(r, err)

	status := httputil.ErrorStatus(err)

	app.render(w, r, status, view, func(data *renderData) {
		switch {
		case errors.Is(err, csrf.ErrEmptyToken):
			data.ErrorMessage = "Empty CSRF token"

		case errors.Is(err, csrf.ErrInvalidToken):
			data.ErrorMessage = "Invalid CSRF token"

		default:
			data.ErrorMessage = "An error has occurred"
		}

		if dataFunc != nil {
			dataFunc(data)
		}
	})

	return true
}

func (app *App) renderError(w http.ResponseWriter, r *http.Request, err error) bool {
	return app.renderErrorView(w, r, err, "error", nil)
}
