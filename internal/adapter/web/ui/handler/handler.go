package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"strings"
	"sync"

	"github.com/polyscone/tofu/internal/adapter/web/httputil"
	"github.com/polyscone/tofu/internal/adapter/web/passport"
	"github.com/polyscone/tofu/internal/adapter/web/sess"
	"github.com/polyscone/tofu/internal/app"
	"github.com/polyscone/tofu/internal/pkg/csrf"
	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/http/router"
	"github.com/polyscone/tofu/internal/pkg/rate"
	"github.com/polyscone/tofu/internal/pkg/session"
	"github.com/polyscone/tofu/internal/pkg/smtp"
)

type emailContent struct {
	Subject string
	Plain   string
	HTML    string
}

type EmailRecipients struct {
	From    string
	ReplyTo string
	To      []string
	Cc      []string
	Bcc     []string
}

type ViewVarsFunc func(r *http.Request) (Vars, error)

type Handler struct {
	*Tenant
	signInPathName string
	files          fs.FS
	templatesMu    sync.RWMutex
	templates      map[string]*template.Template
	funcs          template.FuncMap
	viewVarsFuncs  map[string]ViewVarsFunc
	mux            *router.ServeMux
	Sessions       *session.Manager
}

func New(mux *router.ServeMux, tenant *Tenant, files fs.FS, signInPathName string) *Handler {
	sessions := session.NewManager(tenant.Repo.Web)
	funcs := template.FuncMap{
		"Add":           tmplAdd,
		"Sub":           tmplSub,
		"Mul":           tmplMul,
		"Div":           tmplDiv,
		"Mod":           tmplMod,
		"Ints":          tmplInts,
		"StatusText":    tmplStatusText,
		"Path":          tmplPath(mux),
		"QueryString":   tmplQueryString,
		"FormatTime":    tmplFormatTime,
		"HasPrefix":     tmplHasPrefix,
		"HasSuffix":     tmplHasSuffix,
		"HasPathPrefix": tmplHasPathPrefix(mux),
		"HasString":     tmplHasString,
		"ToStrings":     tmplToStrings,
		"UnescapeHTML":  tmplUnescapeHTML,
	}

	return &Handler{
		Tenant:         tenant,
		signInPathName: signInPathName,
		files:          files,
		templates:      make(map[string]*template.Template),
		funcs:          funcs,
		viewVarsFuncs:  make(map[string]ViewVarsFunc),
		mux:            mux,
		Sessions:       sessions,
	}
}

func (h *Handler) RenewSession(ctx context.Context) ([]byte, error) {
	if err := csrf.RenewToken(ctx); err != nil {
		return nil, errors.Tracef(err)
	}

	if err := h.Sessions.Renew(ctx); err != nil {
		return nil, errors.Tracef(err)
	}

	return csrf.MaskedToken(ctx), nil
}

func (h *Handler) emptyPassport(ctx context.Context) passport.Passport {
	return passport.New(passport.User{})
}

func (h *Handler) Passport(ctx context.Context) passport.Passport {
	if h.Sessions.GetBool(ctx, sess.IsAwaitingTOTP) {
		return h.emptyPassport(ctx)
	}

	userID := h.Sessions.GetInt(ctx, sess.UserID)
	user, err := h.Repo.Account.FindUserByID(ctx, userID)
	if err != nil {
		return h.emptyPassport(ctx)
	}

	var permissions []string
	for _, role := range user.Roles {
		permissions = append(permissions, role.Permissions...)
	}

	return passport.New(passport.User{
		ID:          user.ID,
		IsSignedIn:  h.Sessions.GetBool(ctx, sess.IsSignedIn),
		Permissions: permissions,
	})
}

func (h *Handler) PassportByEmail(ctx context.Context, email string) (passport.Passport, error) {
	user, err := h.Repo.Account.FindUserByEmail(ctx, email)
	if err != nil {
		return h.emptyPassport(ctx), errors.Tracef(err)
	}

	var permissions []string
	for _, role := range user.Roles {
		permissions = append(permissions, role.Permissions...)
	}

	p := passport.New(passport.User{
		ID:          user.ID,
		IsSignedIn:  h.Sessions.GetBool(ctx, sess.IsSignedIn),
		Permissions: permissions,
	})

	return p, nil
}

func (h *Handler) Path(name string, paramArgPairs ...any) string {
	return h.mux.Path(name, paramArgPairs...)
}

func (h *Handler) PathQuery(r *http.Request, name string, paramArgPairs ...any) string {
	q := r.URL.Query().Encode()
	if q != "" && !strings.HasPrefix(q, "?") {
		q = "?" + q
	}

	return h.mux.Path(name, paramArgPairs...) + q
}

func (h *Handler) SetViewVars(name string, vars ViewVarsFunc) {
	if _, ok := h.viewVarsFuncs[name]; ok {
		panic(fmt.Sprintf("default view vars already set for %q", name))
	}

	h.viewVarsFuncs[name] = vars
}

func (h *Handler) template(name string, patterns ...string) *template.Template {
	h.templatesMu.RLock()

	if tmpl := h.templates[name]; tmpl != nil && !h.Tenant.Dev {
		h.templatesMu.RUnlock()

		return tmpl
	}

	h.templatesMu.RUnlock()

	h.templatesMu.Lock()
	defer h.templatesMu.Unlock()

	tmpl := template.New(name).Option("missingkey=default").Funcs(h.funcs)

	for _, pattern := range patterns {
		tmpl = errors.Must(tmpl.ParseFS(h.files, pattern))
	}

	h.templates[name] = tmpl

	return tmpl
}

func (h *Handler) email(name string) *template.Template {
	return h.template(name, "email/"+name+".tmpl")
}

func (h *Handler) emailContentFunc(name string, dataFunc emailDataFunc) (emailContent, error) {
	data := emailData{
		URL: URL{
			Scheme:   h.Tenant.Scheme,
			Host:     h.Tenant.Host,
			Hostname: h.Tenant.Hostname,
			Port:     h.Tenant.Port,
		},
		App: AppData{
			Name:        app.Name,
			Description: app.Description,
			HasSMS:      h.Tenant.SMS.IsConfigured,
		},
	}

	if dataFunc != nil {
		dataFunc(&data)
	}

	email := h.email(name)

	var content emailContent
	var buf bytes.Buffer

	for _, name := range []string{"subject", "plain", "html"} {
		tmpl := email.Lookup(name)
		if tmpl == nil {
			continue
		}

		buf.Reset()

		if err := tmpl.Execute(&buf, data); err != nil {
			return content, errors.Tracef(err)
		}

		switch name {
		case "subject":
			content.Subject = buf.String()

		case "plain":
			content.Plain = buf.String()

		case "html":
			content.HTML = buf.String()
		}
	}

	return content, nil
}

func (h *Handler) emailContent(name string, vars Vars) (emailContent, error) {
	return h.emailContentFunc(name, func(data *emailData) {
		data.Vars = data.Vars.Merge(vars)
	})
}

func (h *Handler) SendEmail(ctx context.Context, recipients EmailRecipients, name string, vars Vars) error {
	content, err := h.emailContent(name, vars)
	if err != nil {
		return errors.Tracef(err)
	}

	msg := smtp.Msg{
		From:    recipients.From,
		ReplyTo: recipients.ReplyTo,
		To:      recipients.To,
		Cc:      recipients.Cc,
		Bcc:     recipients.Bcc,
		Subject: content.Subject,
		Plain:   content.Plain,
		HTML:    content.HTML,
	}

	return errors.Tracef(h.Tenant.Email.Mailer.Send(ctx, msg))
}

func (h *Handler) SendSMS(ctx context.Context, to, body string) error {
	return errors.Tracef(h.Tenant.SMS.Messager.Send(ctx, h.Tenant.SMS.From, to, body))
}

func (h *Handler) SendTOTPSMS(email, telephone string) error {
	ctx := context.Background()

	user, err := h.Repo.Account.FindUserByEmail(ctx, email)
	if err != nil {
		return errors.Tracef(err)
	}

	totp, err := user.GenerateTOTP()
	if err != nil {
		return errors.Tracef(err)
	}

	if telephone == "" {
		telephone = user.TOTPTelephone
	}

	err = h.SendSMS(ctx, telephone, totp)

	return errors.Tracef(err)
}

func (h *Handler) view(name string) *template.Template {
	return h.template(name, "partial/*.tmpl", "view/"+name+".tmpl", "master.tmpl")
}

func (h *Handler) ViewFunc(w http.ResponseWriter, r *http.Request, status int, name string, dataFunc ViewDataFunc) {
	ctx := r.Context()

	passport := h.Passport(ctx)

	data := ViewData{
		View:   name,
		Status: status,
		CSRF:   CSRF{ctx: ctx},
		Form:   Form{Values: r.PostForm},
		URL: URL{
			Scheme:   h.Tenant.Scheme,
			Host:     h.Tenant.Host,
			Hostname: h.Tenant.Hostname,
			Port:     h.Tenant.Port,
			Path:     template.URL(r.URL.Path),
			Query:    Query{Values: r.URL.Query()},
		},
		App: AppData{
			Name:        app.Name,
			Description: app.Description,
			HasSMS:      h.Tenant.SMS.IsConfigured,
		},
		Session: SessionData{
			// Global session keys
			Flash:          h.Sessions.PopStrings(ctx, sess.Flash),
			FlashImportant: h.Sessions.PopStrings(ctx, sess.FlashImportant),
			Redirect:       h.Sessions.GetString(ctx, sess.Redirect),
			HighlightID:    h.Sessions.PopInt(ctx, sess.HighlightID),

			// Account session keys
			UserID:                   h.Sessions.GetInt(ctx, sess.UserID),
			Email:                    h.Sessions.GetString(ctx, sess.Email),
			TOTPMethod:               h.Sessions.GetString(ctx, sess.TOTPMethod),
			HasActivatedTOTP:         h.Sessions.GetBool(ctx, sess.HasActivatedTOTP),
			IsAwaitingTOTP:           h.Sessions.GetBool(ctx, sess.IsAwaitingTOTP),
			IsSignedIn:               h.Sessions.GetBool(ctx, sess.IsSignedIn),
			PasswordKnownBreachCount: h.Sessions.GetInt(ctx, sess.PasswordKnownBreachCount),
		},
		Passport: passport,
	}

	if vars, ok := h.viewVarsFuncs[name]; ok {
		defaults, err := vars(r)
		if h.ErrorView(w, r, errors.Tracef(err), "error", nil) {
			return
		}

		data.Vars = data.Vars.Merge(defaults)
	}

	if dataFunc != nil {
		dataFunc(&data)
	}

	// Make sure the current view name isn't overwritten by a user function
	data.View = name

	var buf bytes.Buffer

	if err := h.view(name).ExecuteTemplate(&buf, "master", data); err != nil {
		httputil.LogError(r, err)

		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)

		return
	}

	w.Header().Set("content-type", "text/html")
	w.WriteHeader(status)

	if _, err := buf.WriteTo(w); err != nil {
		httputil.LogError(r, errors.Tracef(err))
	}
}

func (h *Handler) View(w http.ResponseWriter, r *http.Request, status int, name string, vars Vars) {
	h.ViewFunc(w, r, status, name, func(data *ViewData) {
		data.Vars = data.Vars.Merge(vars)
	})
}

func (h *Handler) ErrorViewFunc(w http.ResponseWriter, r *http.Request, err error, name string, dataFunc ViewDataFunc) bool {
	if err == nil {
		return false
	}

	httputil.LogError(r, errors.Tracef(err))

	status := httputil.ErrorStatus(err)

	h.ViewFunc(w, r, status, name, func(data *ViewData) {
		switch {
		case errors.Is(err, httputil.ErrNotFound):
			data.ErrorMessage = "The page you were looking for could not be found."

		case errors.Is(err, httputil.ErrMethodNotAllowed):
			data.ErrorMessage = "Method not allowed."

		case errors.Is(err, httputil.ErrForbidden),
			errors.Is(err, app.ErrForbidden):

			data.ErrorMessage = "You do not have permission to access this resource."

		case errors.Is(err, http.ErrHandlerTimeout):
			data.ErrorMessage = "The server took too long to respond."

		case errors.Is(err, app.ErrUnauthorised):
			data.ErrorMessage = "You do not have sufficient permissions."

		case errors.Is(err, app.ErrMalformedInput),
			errors.Is(err, app.ErrInvalidInput),
			errors.Is(err, app.ErrConflictingInput):

			data.ErrorMessage = "Invalid input."

			if trace, ok := err.(errors.Trace); ok {
				data.Errors = trace.Fields()
			}

		case errors.Is(err, csrf.ErrEmptyToken):
			data.ErrorMessage = "Empty CSRF token."

		case errors.Is(err, csrf.ErrInvalidToken):
			data.ErrorMessage = "Invalid CSRF token."

		case errors.Is(err, rate.ErrInsufficientTokens):
			data.ErrorMessage = "You have made too many consecutive requests."

		default:
			data.ErrorMessage = "An error has occurred."
		}

		if dataFunc != nil {
			dataFunc(data)
		}
	})

	return true
}

func (h *Handler) ErrorView(w http.ResponseWriter, r *http.Request, err error, name string, vars Vars) bool {
	return h.ErrorViewFunc(w, r, errors.Tracef(err), name, func(data *ViewData) {
		data.Vars = data.Vars.Merge(vars)
	})
}

func (h *Handler) ErrorJSON(w http.ResponseWriter, r *http.Request, err error) bool {
	if err == nil {
		return false
	}

	httputil.LogError(r, errors.Tracef(err))

	var displayOK bool
	status := httputil.ErrorStatus(err)

	switch {
	case errors.Is(err, httputil.ErrBadJSON):
		status, displayOK = http.StatusBadRequest, true

	case errors.Is(err, httputil.ErrExpectedJSON):
		status, displayOK = http.StatusUnsupportedMediaType, true

	default:
		switch {
		case errors.Is(err, http.ErrHandlerTimeout),
			errors.Is(err, app.ErrMalformedInput),
			errors.Is(err, app.ErrInvalidInput),
			errors.Is(err, app.ErrUnauthorised),
			errors.Is(err, csrf.ErrEmptyToken),
			errors.Is(err, csrf.ErrInvalidToken):

			displayOK = true
		}
	}

	w.Header().Set("content-type", "application/json")
	w.WriteHeader(status)

	detail := map[string]any{"error": strings.ToLower(http.StatusText(status))}
	if displayOK && 400 <= status && status <= 499 {
		detail["error"] = err.Error()

		if trace, ok := err.(errors.Trace); ok {
			fields := trace.Fields()

			if fields != nil {
				detail["fields"] = fields
			}
		}
	}

	if err := json.NewEncoder(w).Encode(detail); err != nil {
		httputil.LogError(r, errors.Tracef(err))
	}

	return true
}

func (h *Handler) JSON(w http.ResponseWriter, r *http.Request, data any) bool {
	w.Header().Set("content-type", "application/json")

	return !h.ErrorJSON(w, r, errors.Tracef(json.NewEncoder(w).Encode(data)))
}

func (h *Handler) AddFlashf(ctx context.Context, format string, a ...any) {
	flash := h.Sessions.GetStrings(ctx, sess.Flash)

	flash = append(flash, fmt.Sprintf(format, a...))

	h.Sessions.Set(ctx, sess.Flash, flash)
}

func (h *Handler) AddFlashImportantf(ctx context.Context, format string, a ...any) {
	flash := h.Sessions.GetStrings(ctx, sess.FlashImportant)

	flash = append(flash, fmt.Sprintf(format, a...))

	h.Sessions.Set(ctx, sess.FlashImportant, flash)
}

type PredicateFunc func(p passport.Passport) bool

func (h *Handler) RequireSignIn(w http.ResponseWriter, r *http.Request) bool {
	ctx := r.Context()
	passport := h.Passport(ctx)

	if !passport.IsSignedIn() {
		h.Sessions.Set(ctx, sess.Redirect, r.URL.String())

		http.Redirect(w, r, h.mux.Path(h.signInPathName), http.StatusSeeOther)

		return false
	}

	return true
}

func (h *Handler) RequireAuth(check PredicateFunc) router.BeforeHookFunc {
	return func(w http.ResponseWriter, r *http.Request) bool {
		ctx := r.Context()
		passport := h.Passport(ctx)

		if !passport.IsSignedIn() {
			h.Sessions.Set(ctx, sess.Redirect, r.URL.String())

			http.Redirect(w, r, h.mux.Path(h.signInPathName), http.StatusSeeOther)

			return false
		}

		if !check(passport) {
			h.ErrorView(w, r, errors.Tracef(app.ErrUnauthorised), "error", nil)

			return false
		}

		return true
	}
}
