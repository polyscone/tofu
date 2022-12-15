package ui

import (
	"bytes"
	"embed"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"text/template"

	"github.com/polyscone/tofu/internal/adapter/web/internal/httputil"
	"github.com/polyscone/tofu/internal/pkg/command"
	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/http/router"
	"github.com/polyscone/tofu/internal/pkg/session"
)

//go:embed "files/static" "files/template"
var embeddedFiles embed.FS

var tmplFuncs = template.FuncMap{
	"StatusText": http.StatusText,
}

type App struct {
	dev         bool
	bus         command.Bus
	sessions    *session.Manager
	files       fs.FS
	templatesMu sync.RWMutex
	templates   map[string]*template.Template
}

func New(bus command.Bus, sessions *session.Manager) *App {
	files := fs.FS(embeddedFiles)

	var dev bool
	dir := "internal/web/internal/ui"
	if info, err := os.Stat(dir); err == nil && info.IsDir() {
		dev = true
		files = os.DirFS(dir)
	}

	templates := make(map[string]*template.Template)

	return &App{
		dev:       dev,
		bus:       bus,
		sessions:  sessions,
		files:     files,
		templates: templates,
	}
}

func (app *App) Routes() http.Handler {
	static := errors.Must(fs.Sub(app.files, "files"))

	mux := router.NewServeMux()

	mux.GetHandler("/static/:rest", http.FileServer(http.FS(static)))
	mux.Get("/favicon.ico", app.faviconGet)
	mux.Get("/robots.txt", app.robotsGet)

	mux.Get("/", app.homeGet)

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

	tmpl := template.New(key).Funcs(tmplFuncs)
	tmpl = errors.Must(tmpl.ParseFS(app.files, "files/template/master.go.html"))
	tmpl = errors.Must(tmpl.ParseFS(app.files, "files/template/partial/*.go.html"))
	tmpl = errors.Must(tmpl.ParseFS(app.files, "files/template/view/"+view+".go.html"))

	app.templates[key] = tmpl

	return tmpl
}

func (app *App) render(w http.ResponseWriter, r *http.Request, status int, view string) {
	var buf bytes.Buffer

	data := struct {
		Status int
	}{
		Status: status,
	}

	if err := app.view(view).ExecuteTemplate(&buf, "master", data); err != nil {
		httputil.LogError(r, errors.Tracef(err))

		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)

		return
	}

	w.Header().Set("content-type", "text/html")
	w.WriteHeader(status)

	buf.WriteTo(w)
}

func (app *App) renderError(w http.ResponseWriter, r *http.Request, err error) bool {
	if err == nil {
		return false
	}

	httputil.LogError(r, err)

	status := httputil.ErrorStatus(err)

	app.render(w, r, status, "error")

	return true
}
