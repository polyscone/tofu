package ui

import (
	"bytes"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"path"
	"time"

	"github.com/polyscone/tofu/internal/adapter/web/guard"
	"github.com/polyscone/tofu/internal/adapter/web/handler"
	"github.com/polyscone/tofu/internal/adapter/web/httputil"
	"github.com/polyscone/tofu/internal/adapter/web/sess"
	"github.com/polyscone/tofu/internal/app"
	"github.com/polyscone/tofu/internal/app/account"
	"github.com/polyscone/tofu/internal/app/system"
	"github.com/polyscone/tofu/internal/pkg/errsx"
)

type ViewData struct {
	View         string
	ContentType  string
	Status       int
	CSRF         handler.CSRF
	ErrorMessage string
	Errors       errsx.Map
	Now          time.Time
	Form         handler.Form
	URL          handler.URL
	App          handler.AppData
	Session      handler.SessionData
	Config       *system.Config
	User         *account.User
	Passport     guard.Passport
	ComData      map[string]any
	Vars         handler.Vars
}

func (v ViewData) WithComData(pairs ...any) (ViewData, error) {
	if len(pairs)%2 == 1 {
		return v, errors.New("WithComData: want key value pairs")
	}

	v.ComData = make(map[string]any, len(pairs)/2)
	for i := 0; i < len(pairs); i += 2 {
		key := fmt.Sprintf("%v", pairs[i])
		value := pairs[i+1]

		if key == "ComData" {
			comData, ok := value.(map[string]any)
			if ok {
				for key, value := range comData {
					v.ComData[key] = value
				}

				continue
			}
		}

		v.ComData[key] = value
	}

	return v, nil
}

type ViewDataFunc func(data *ViewData)

type Renderer struct {
	h           *Handler
	contentType string
}

func NewRenderer(h *Handler, contentType string) *Renderer {
	return &Renderer{
		h:           h,
		contentType: contentType,
	}
}

func (rn *Renderer) ViewFunc(w http.ResponseWriter, r *http.Request, status int, view string, dataFunc ViewDataFunc) {
	ctx := r.Context()
	config := rn.h.Config(ctx)
	user := rn.h.User(ctx)
	passport := rn.h.Passport(ctx)

	data := ViewData{
		View:        view,
		ContentType: rn.contentType,
		Status:      status,
		CSRF:        handler.CSRF{Ctx: ctx},
		Now:         time.Now(),
		Form:        handler.Form{Values: r.PostForm},
		URL: handler.URL{
			Scheme: rn.h.Tenant.Scheme,
			Host:   rn.h.Tenant.Host,
			Path:   template.URL(r.URL.Path),
			Query:  handler.Query{Values: r.URL.Query()},
		},
		App: handler.AppData{
			Name:        app.Name,
			ShortName:   app.ShortName,
			Description: app.Description,
			ThemeColour: app.ThemeColour,
		},
		Session: handler.SessionData{
			// Global session keys
			Flash:          rn.h.Sessions.PopStrings(ctx, sess.Flash),
			FlashImportant: rn.h.Sessions.PopStrings(ctx, sess.FlashImportant),
			FlashError:     rn.h.Sessions.PopStrings(ctx, sess.FlashError),
			Redirect:       rn.h.Sessions.GetString(ctx, sess.Redirect),
			HighlightID:    rn.h.Sessions.PopInt(ctx, sess.HighlightID),

			// Account session keys
			UserID:                   rn.h.Sessions.GetInt(ctx, sess.UserID),
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
	}

	if vars, ok := rn.h.viewVarsFuncs[view]; ok {
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

	dir := path.Dir(view)
	tmpl := rn.h.Template(templateFiles, rn.h.funcs, view,
		"partial/*.tmpl",
		"view/"+dir+"/com_*.tmpl",
		"view/"+view+".tmpl",
		"master/*.tmpl",
	)

	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, "master", data); err != nil {
		rn.h.Logger(ctx).Error("execute view template", "error", err)

		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)

		return
	}

	w.Header().Set("content-type", rn.contentType)
	w.WriteHeader(status)

	if _, err := buf.WriteTo(w); err != nil {
		rn.h.Logger(ctx).Error("write view template response", "error", err)
	}
}

func (rn *Renderer) View(w http.ResponseWriter, r *http.Request, status int, view string, vars handler.Vars) {
	rn.ViewFunc(w, r, status, view, func(data *ViewData) {
		data.Vars = data.Vars.Merge(vars)
	})
}

func (rn *Renderer) Handler(view string) http.HandlerFunc {
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
			errors.Is(err, app.ErrConflictingInput):

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

func (rn *Renderer) ErrorView(w http.ResponseWriter, r *http.Request, msg string, err error, view string, vars handler.Vars) {
	rn.ErrorViewFunc(w, r, msg, err, view, func(data *ViewData) {
		data.Vars = data.Vars.Merge(vars)
	})
}
