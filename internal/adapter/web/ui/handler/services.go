package handler

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"net/url"
	"sync"

	"github.com/polyscone/tofu/internal/adapter/web/httputil"
	"github.com/polyscone/tofu/internal/adapter/web/passport"
	"github.com/polyscone/tofu/internal/adapter/web/sess"
	"github.com/polyscone/tofu/internal/adapter/web/smtp"
	"github.com/polyscone/tofu/internal/app"
	"github.com/polyscone/tofu/internal/pkg/command"
	"github.com/polyscone/tofu/internal/pkg/csrf"
	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/http/router"
	"github.com/polyscone/tofu/internal/pkg/session"
	"github.com/polyscone/tofu/internal/port"
	"github.com/polyscone/tofu/internal/port/account"
)

type CSRF struct {
	ctx context.Context
}

func (c CSRF) Token() string {
	return base64.RawURLEncoding.EncodeToString(csrf.MaskedToken(c.ctx))
}

type AppData struct {
	Name        string
	Description string
}

type SessionData struct {
	UserID          string
	Email           string
	HasVerifiedTOTP bool
	IsAwaitingTOTP  bool
	IsAuthenticated bool
}

type Vars map[string]any

func (v Vars) Merge(rhs Vars) Vars {
	if v == nil {
		v = make(Vars, len(rhs))
	}

	for key, value := range rhs {
		v[key] = value
	}

	return v
}

type Data struct {
	Status       int
	CSRF         CSRF
	ErrorMessage string
	Errors       errors.Map
	Form         url.Values
	Query        url.Values
	App          AppData
	Session      SessionData
	Vars         Vars
}

type DataFunc func(data *Data)

type Options struct {
	Cache bool
	Files fs.FS
	Funcs template.FuncMap
}

type Services struct {
	cache       bool
	files       fs.FS
	templatesMu sync.RWMutex
	templates   map[string]*template.Template
	funcs       template.FuncMap
	defaultVars map[string]Vars
	mux         *router.ServeMux
	Bus         command.Bus
	Sessions    *session.Manager
	Mailer      smtp.Mailer
}

func NewServices(mux *router.ServeMux, bus command.Bus, sessions *session.Manager, mailer smtp.Mailer, opts Options) *Services {
	svc := Services{
		cache:       opts.Cache,
		files:       opts.Files,
		templates:   make(map[string]*template.Template),
		funcs:       make(template.FuncMap),
		defaultVars: make(map[string]Vars),
		mux:         mux,
		Bus:         bus,
		Sessions:    sessions,
		Mailer:      mailer,
	}

	for name, fn := range opts.Funcs {
		svc.funcs[name] = fn
	}

	return &svc
}

func (svc *Services) Passport(ctx context.Context) passport.Passport {
	if svc.Sessions.GetBool(ctx, sess.IsAwaitingTOTP) {
		return passport.Empty
	}

	userID := svc.Sessions.GetString(ctx, sess.UserID)
	cmd := account.FindAuthInfo{
		UserID: userID,
	}
	info, err := cmd.Execute(ctx, svc.Bus)
	if err != nil {
		return passport.Empty
	}

	return passport.New(ctx, svc.Sessions, userID, info.Claims, info.Roles, info.Permissions)
}

func (svc *Services) PassportByEmail(ctx context.Context, email string) (passport.Passport, error) {
	cmd := account.FindUserByEmail{
		Email: email,
	}
	user, err := cmd.Execute(ctx, svc.Bus)
	if err != nil {
		return passport.Empty, errors.Tracef(err)
	}

	return passport.New(ctx, svc.Sessions, user.ID, nil, nil, nil), nil
}

func (svc *Services) Path(name string, paramArgPairs ...string) string {
	return svc.mux.Path(name, paramArgPairs...)
}

func (svc *Services) SetDefaultVars(view string, vars Vars) {
	if _, ok := svc.defaultVars[view]; ok {
		panic(fmt.Sprintf("default vars already set for %q", view))
	}

	svc.defaultVars[view].Merge(vars)
}

func (svc *Services) view(view string) *template.Template {
	svc.templatesMu.RLock()

	if tmpl := svc.templates[view]; tmpl != nil && svc.cache {
		svc.templatesMu.RUnlock()

		return tmpl
	}

	svc.templatesMu.RUnlock()

	svc.templatesMu.Lock()
	defer svc.templatesMu.Unlock()

	tmpl := template.New(view).Option("missingkey=zero").Funcs(svc.funcs)
	tmpl = errors.Must(tmpl.ParseFS(svc.files, "master.go.html"))
	tmpl = errors.Must(tmpl.ParseFS(svc.files, "partial/*.go.html"))
	tmpl = errors.Must(tmpl.ParseFS(svc.files, "view/"+view+".go.html"))

	svc.templates[view] = tmpl

	return tmpl
}

func (svc *Services) RenderFunc(w http.ResponseWriter, r *http.Request, status int, view string, dataFunc DataFunc) {
	var buf bytes.Buffer

	ctx := r.Context()

	data := Data{
		Status: status,
		CSRF:   CSRF{ctx: ctx},
		Form:   r.PostForm,
		Query:  r.URL.Query(),
		App: AppData{
			Name:        app.Name,
			Description: app.Description,
		},
		Session: SessionData{
			UserID:          svc.Sessions.GetString(ctx, sess.UserID),
			Email:           svc.Sessions.GetString(ctx, sess.Email),
			HasVerifiedTOTP: svc.Sessions.GetBool(ctx, sess.HasVerifiedTOTP),
			IsAwaitingTOTP:  svc.Sessions.GetBool(ctx, sess.IsAwaitingTOTP),
			IsAuthenticated: svc.Sessions.GetBool(ctx, sess.IsAuthenticated),
		},
	}

	if vars, ok := svc.defaultVars[view]; ok {
		data.Vars.Merge(vars)
	}

	if dataFunc != nil {
		dataFunc(&data)
	}

	if err := svc.view(view).ExecuteTemplate(&buf, "master", data); err != nil {
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

func (svc *Services) Render(w http.ResponseWriter, r *http.Request, status int, view string, vars Vars) {
	svc.RenderFunc(w, r, status, view, func(data *Data) {
		data.Vars.Merge(vars)
	})
}

func (svc *Services) RenderError(w http.ResponseWriter, r *http.Request, err error, view string, dataFunc DataFunc) bool {
	if err == nil {
		return false
	}

	httputil.LogError(r, errors.Tracef(err))

	status := httputil.ErrorStatus(err)

	svc.RenderFunc(w, r, status, view, func(data *Data) {
		switch {
		case errors.Is(err, port.ErrInvalidInput):
			data.ErrorMessage = "Invalid input"

			if trace, ok := err.(errors.Trace); ok {
				data.Errors = trace.Fields()
			}

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
