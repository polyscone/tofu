package handler

import (
	"bytes"
	"errors"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"time"

	"github.com/polyscone/tofu/internal/app"
	"github.com/polyscone/tofu/internal/app/account"
	"github.com/polyscone/tofu/internal/app/system"
	"github.com/polyscone/tofu/internal/pkg/errsx"
	"github.com/polyscone/tofu/internal/web/guard"
	"github.com/polyscone/tofu/internal/web/httputil"
	"github.com/polyscone/tofu/internal/web/sess"
)

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
	View         string
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

type ViewDataFunc func(data *ViewData)
type ViewVarsFunc func(r *http.Request) (Vars, error)
type TemplatePathsFunc func(view string) []string
type TemplateProcessFunc func(w http.ResponseWriter, r *http.Request, template *bytes.Buffer) []byte

type Renderer struct {
	h             *Handler
	templateFiles fs.FS
	templatePaths TemplatePathsFunc
	funcs         template.FuncMap
	viewVarsFuncs map[string]ViewVarsFunc
	process       TemplateProcessFunc
}

func NewRenderer(h *Handler, templateFiles fs.FS, templatePaths TemplatePathsFunc, funcs template.FuncMap, process TemplateProcessFunc) *Renderer {
	return &Renderer{
		h:             h,
		templateFiles: templateFiles,
		templatePaths: templatePaths,
		funcs:         funcs,
		viewVarsFuncs: make(map[string]ViewVarsFunc),
		process:       process,
	}
}

func (rn *Renderer) ViewFunc(w http.ResponseWriter, r *http.Request, status int, view string, dataFunc ViewDataFunc) {
	ctx := r.Context()
	config := rn.h.Config(ctx)
	user := rn.h.User(ctx)
	passport := rn.h.Passport(ctx)

	data := ViewData{
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
		},
		Session: SessionData{
			// Global session keys
			Flash:          rn.h.Sessions.PopStrings(ctx, sess.Flash),
			FlashImportant: rn.h.Sessions.PopStrings(ctx, sess.FlashImportant),
			FlashError:     rn.h.Sessions.PopStrings(ctx, sess.FlashError),
			Redirect:       rn.h.Sessions.GetString(ctx, sess.Redirect),
			HighlightID:    rn.h.Sessions.PopString(ctx, sess.HighlightID),

			// Account session keys
			UserID:                   rn.h.Sessions.GetString(ctx, sess.UserID),
			Email:                    rn.h.Sessions.GetString(ctx, sess.Email),
			TOTPMethod:               rn.h.Sessions.GetString(ctx, sess.TOTPMethod),
			HasActivatedTOTP:         rn.h.Sessions.GetBool(ctx, sess.HasActivatedTOTP),
			IsAwaitingTOTP:           rn.h.Sessions.GetBool(ctx, sess.IsAwaitingTOTP),
			IsSignedIn:               rn.h.Sessions.GetBool(ctx, sess.IsSignedIn),
			KnownPasswordBreachCount: rn.h.Sessions.GetInt(ctx, sess.KnownPasswordBreachCount),
		},
		Config:   config,
		User:     user,
		Passport: passport,
		State:    &State{},
	}

	if vars, ok := rn.viewVarsFuncs[view]; ok {
		defaults, err := vars(r)
		if err != nil {
			rn.ErrorView(w, r, "vars", err, "site/error", nil)

			return
		}

		data.Vars = data.Vars.Merge(defaults)
	}

	if dataFunc != nil {
		dataFunc(&data)
	}

	// Make sure the current view name isn't overwritten by a user function
	data.View = view

	tmpl := rn.h.Template(rn.templateFiles, rn.templatePaths(view), rn.funcs, view)

	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, "master", data); err != nil {
		rn.h.Logger(ctx).Error("execute view template", "error", err)

		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)

		return
	}

	if rn.process != nil {
		b := rn.process(w, r, &buf)
		if b != nil {
			buf = *bytes.NewBuffer(b)
		}
	}

	w.WriteHeader(status)

	if _, err := buf.WriteTo(w); err != nil {
		rn.h.Logger(ctx).Error("write view template response", "error", err)
	}
}

func (rn *Renderer) View(w http.ResponseWriter, r *http.Request, status int, view string, vars Vars) {
	rn.ViewFunc(w, r, status, view, func(data *ViewData) {
		data.Vars = data.Vars.Merge(vars)
	})
}

func (rn *Renderer) SetViewVars(name string, vars ViewVarsFunc) {
	if _, ok := rn.viewVarsFuncs[name]; ok {
		panic(fmt.Sprintf("default view vars already set for %q", name))
	}

	rn.viewVarsFuncs[name] = vars
}

func (rn *Renderer) HandlerFunc(view string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rn.View(w, r, http.StatusOK, view, nil)
	}
}

func (rn *Renderer) ErrorViewFunc(w http.ResponseWriter, r *http.Request, msg string, err error, view string, dataFunc ViewDataFunc) {
	ctx := r.Context()

	rn.h.Logger(ctx).Error(msg, "error", err)

	status := httputil.ErrorStatus(err)

	rn.ViewFunc(w, r, status, view, func(data *ViewData) {
		data.ErrorMessage = httputil.ErrorMessage(err)

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
			dataFunc(data)
		}
	})
}

func (rn *Renderer) ErrorView(w http.ResponseWriter, r *http.Request, msg string, err error, view string, vars Vars) {
	rn.ErrorViewFunc(w, r, msg, err, view, func(data *ViewData) {
		data.Vars = data.Vars.Merge(vars)
	})
}
