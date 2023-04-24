package ui

import (
	"bytes"
	"embed"
	"encoding/base64"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
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

type Option func(ui *UI)

func WithDev(value bool) Option {
	return func(ui *UI) {
		ui.dev = value
	}
}

type UI struct {
	dev         bool
	bus         command.Bus
	sessions    *session.Manager
	tokens      token.Repo
	mailer      smtp.Mailer
	files       fs.FS
	templatesMu sync.RWMutex
	templates   map[string]*template.Template
}

func New(bus command.Bus, sessions *session.Manager, tokens token.Repo, mailer smtp.Mailer, opts ...Option) *UI {
	files := fs.FS(embeddedFiles)
	templates := make(map[string]*template.Template)

	ui := UI{
		bus:       bus,
		sessions:  sessions,
		tokens:    tokens,
		mailer:    mailer,
		files:     files,
		templates: templates,
	}

	for _, opt := range opts {
		opt(&ui)
	}

	if ui.dev {
		dir := "internal/adapter/web/internal/ui"
		if info, err := os.Stat(dir); err == nil && info.IsDir() {
			ui.files = fstack.New(os.DirFS(dir), ui.files)
		}
	}

	return &ui
}

func (ui *UI) Routes() http.Handler {
	static := errors.Must(fs.Sub(ui.files, "files/static"))

	mux := router.NewServeMux()

	mux.Redirect(http.MethodGet, "/favicon.ico", "/favicon.png", http.StatusTemporaryRedirect)

	mux.Get("/", ui.homeGet)

	mux.Prefix("/account", func(mux *router.ServeMux) {
		mux.Get("/activate", ui.accountActivateGet)
		mux.Post("/activate", ui.accountActivatePost)

		mux.Get("/register", ui.accountRegisterGet)
		mux.Post("/register", ui.accountRegisterPost)

		mux.Get("/login", ui.accountLoginGet)
		mux.Post("/login", ui.accountLoginPost)

		mux.Post("/logout", ui.accountLogoutPost)

		mux.Get("/forgotten-password", ui.accountForgottenPasswordGet)
		mux.Post("/forgotten-password", ui.accountForgottenPasswordPost)
		mux.Put("/forgotten-password", ui.accountForgottenPasswordPut)
	})

	mux.GetHandler("/:rest", http.FileServer(http.FS(static)))

	mux.NotFound(func(w http.ResponseWriter, r *http.Request) {
		ui.renderError(w, r, errors.Tracef("%w: %v %v", httputil.ErrNotFound, r.Method, r.URL))
	})

	mux.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
		ui.renderError(w, r, errors.Tracef("%w: %v %v", httputil.ErrMethodNotAllowed, r.Method, r.URL))
	})

	return mux
}

func (ui *UI) ErrorHandler(w http.ResponseWriter, r *http.Request, err error) {
	ui.renderError(w, r, errors.Tracef(err))
}

func (ui *UI) csrfToken(r *http.Request) string {
	ctx := r.Context()

	return base64.RawURLEncoding.EncodeToString(csrf.MaskedToken(ctx))
}

func (ui *UI) view(view string) *template.Template {
	ui.templatesMu.RLock()

	// Return the cached template only when we're not in a dev environment
	if tmpl := ui.templates[view]; tmpl != nil && !ui.dev {
		ui.templatesMu.RUnlock()

		return tmpl
	}

	ui.templatesMu.RUnlock()

	ui.templatesMu.Lock()
	defer ui.templatesMu.Unlock()

	key := strings.TrimSuffix(filepath.Base(view), ".go.html")

	tmpl := template.New(key).Option("missingkey=zero").Funcs(tmplFuncs)
	tmpl = errors.Must(tmpl.ParseFS(ui.files, "files/template/master.go.html"))
	tmpl = errors.Must(tmpl.ParseFS(ui.files, "files/template/partial/*.go.html"))
	tmpl = errors.Must(tmpl.ParseFS(ui.files, "files/template/view/"+view+".go.html"))

	ui.templates[key] = tmpl

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

func (ui *UI) render(w http.ResponseWriter, r *http.Request, status int, view string, dataFunc renderDataFunc) {
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
		CSRFToken: ui.csrfToken(r),
		Status:    status,
		PostForm:  postForm,
		Query:     query,
		Session: sessionRenderData{
			UserID:         ui.sessions.GetString(ctx, sesskey.UserID),
			Email:          ui.sessions.GetString(ctx, sesskey.Email),
			IsAwaitingTOTP: ui.sessions.GetBool(ctx, sesskey.IsAwaitingTOTP),
		},
	}

	if dataFunc != nil {
		dataFunc(&data)
	}

	if err := ui.view(view).ExecuteTemplate(&buf, "master", data); err != nil {
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

func (ui *UI) renderErrorView(w http.ResponseWriter, r *http.Request, err error, view string, dataFunc renderDataFunc) bool {
	if err == nil {
		return false
	}

	httputil.LogError(r, err)

	status := httputil.ErrorStatus(err)

	ui.render(w, r, status, view, func(data *renderData) {
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

func (ui *UI) renderError(w http.ResponseWriter, r *http.Request, err error) bool {
	return ui.renderErrorView(w, r, err, "error", nil)
}

var matchFirstUpper = regexp.MustCompile("(.)([A-Z][a-z]+)")
var matchAllUppers = regexp.MustCompile("([a-z0-9])([A-Z])")

func toKebabCase(str string) string {
	kebab := matchFirstUpper.ReplaceAllString(str, "${1}-${2}")
	kebab = matchAllUppers.ReplaceAllString(kebab, "${1}-${2}")

	return strings.ToLower(kebab)
}

func decodeForm(r *http.Request, dst any) error {
	value := reflect.ValueOf(dst)
	if value.Kind() != reflect.Ptr {
		panic(fmt.Sprintf("want pointer to a struct; got %T", dst))
	}

	s := value.Elem()
	if s.Kind() != reflect.Struct {
		panic(fmt.Sprintf("want pointer to a struct; got %T", dst))
	}

	for i := 0; i < s.NumField(); i++ {
		typeField := s.Type().Field(i)

		tag := typeField.Tag.Get("form")
		if tag == "" {
			tag = toKebabCase(typeField.Name)
		}

		str := r.PostFormValue(tag)
		field := s.Field(i)

		switch typeField.Type.Kind() {
		case reflect.Bool:
			field.SetBool(str == "1" || str == "checked")

		case reflect.Float32:
			value, err := strconv.ParseFloat(str, 32)
			if err != nil {
				return errors.Tracef(err)
			}

			field.SetFloat(value)

		case reflect.Float64:
			value, err := strconv.ParseFloat(str, 64)
			if err != nil {
				return errors.Tracef(err)
			}

			field.SetFloat(value)

		case reflect.Int8:
			value, err := strconv.ParseInt(str, 10, 8)
			if err != nil {
				return errors.Tracef(err)
			}

			field.SetInt(value)

		case reflect.Int16:
			value, err := strconv.ParseInt(str, 10, 16)
			if err != nil {
				return errors.Tracef(err)
			}

			field.SetInt(value)

		case reflect.Int32:
			value, err := strconv.ParseInt(str, 10, 32)
			if err != nil {
				return errors.Tracef(err)
			}

			field.SetInt(value)

		case reflect.Int64:
			value, err := strconv.ParseInt(str, 10, 64)
			if err != nil {
				return errors.Tracef(err)
			}

			field.SetInt(value)

		case reflect.Int:
			value, err := strconv.ParseInt(str, 10, 64)
			if err != nil {
				return errors.Tracef(err)
			}

			field.SetInt(value)

		case reflect.Uint8:
			value, err := strconv.ParseUint(str, 10, 8)
			if err != nil {
				return errors.Tracef(err)
			}

			field.SetUint(value)

		case reflect.Uint16:
			value, err := strconv.ParseUint(str, 10, 16)
			if err != nil {
				return errors.Tracef(err)
			}

			field.SetUint(value)

		case reflect.Uint32:
			value, err := strconv.ParseUint(str, 10, 32)
			if err != nil {
				return errors.Tracef(err)
			}

			field.SetUint(value)

		case reflect.Uint64:
			value, err := strconv.ParseUint(str, 10, 64)
			if err != nil {
				return errors.Tracef(err)
			}

			field.SetUint(value)

		case reflect.Uint:
			value, err := strconv.ParseUint(str, 10, 64)
			if err != nil {
				return errors.Tracef(err)
			}

			field.SetUint(value)

		case reflect.String:
			field.SetString(str)

		default:
			panic(fmt.Sprintf("unsupported struct field type %q", typeField.Type.Kind()))
		}
	}

	return nil
}
