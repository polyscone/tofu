package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/polyscone/tofu/internal/adapter/web/guard"
	"github.com/polyscone/tofu/internal/adapter/web/httputil"
	"github.com/polyscone/tofu/internal/adapter/web/sess"
	"github.com/polyscone/tofu/internal/app"
	"github.com/polyscone/tofu/internal/app/account"
	"github.com/polyscone/tofu/internal/app/system"
	"github.com/polyscone/tofu/internal/pkg/csrf"
	"github.com/polyscone/tofu/internal/pkg/errsx"
	"github.com/polyscone/tofu/internal/pkg/http/router"
	"github.com/polyscone/tofu/internal/pkg/rate"
	"github.com/polyscone/tofu/internal/pkg/realip"
	"github.com/polyscone/tofu/internal/pkg/session"
	"github.com/polyscone/tofu/internal/pkg/sms"
	"github.com/polyscone/tofu/internal/pkg/smtp"
	"github.com/polyscone/tofu/internal/pkg/uuid"
	"github.com/polyscone/tofu/internal/repository"
	"golang.org/x/exp/slog"
)

type ctxKey int

const (
	ctxLogger ctxKey = iota
	ctxConfig
	ctxUser
	ctxPassport
)

var httpClient = http.Client{Timeout: 10 * time.Second}

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
	signInPathName       string
	systemConfigPathName string
	files                fs.FS
	templatesMu          sync.RWMutex
	templates            map[string]*template.Template
	funcs                template.FuncMap
	viewVarsFuncs        map[string]ViewVarsFunc
	mux                  *router.ServeMux
	Sessions             *session.Manager
}

func New(mux *router.ServeMux, tenant *Tenant, files fs.FS, signInPathName, systemConfigPathName string) *Handler {
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
		"Join":          tmplJoin,
		"UnescapeHTML":  tmplUnescapeHTML,
	}

	return &Handler{
		Tenant:               tenant,
		signInPathName:       signInPathName,
		systemConfigPathName: systemConfigPathName,
		files:                files,
		templates:            make(map[string]*template.Template),
		funcs:                funcs,
		viewVarsFuncs:        make(map[string]ViewVarsFunc),
		mux:                  mux,
		Sessions:             sessions,
	}
}

func (h *Handler) Middleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		requestID, err := uuid.NewV4()
		if err != nil {
			h.Log.Error("handler middleware: new v4 UUID", "error", err)

			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)

			return
		}

		config, err := h.Repo.System.FindConfig(ctx)
		if err != nil {
			h.Log.Error("handler middleware: find config", "error", err)

			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)

			return
		}

		user := &account.User{}
		userID := h.Sessions.GetInt(ctx, sess.UserID)
		isAwaitingTOTP := h.Sessions.GetBool(ctx, sess.IsAwaitingTOTP)
		if userID != 0 {
			var err error
			user, err = h.Repo.Account.FindUserByID(ctx, userID)
			if err != nil && !errors.Is(err, repository.ErrNotFound) {
				h.Log.Error("handler middleware: find user by id", "error", err)
			}
		}

		var passport guard.Passport
		if user.ID == 0 || isAwaitingTOTP {
			passport = guard.NewPassport(config.RequiresSetup, guard.User{})
		} else {
			passport = guard.NewPassport(config.RequiresSetup, guard.User{
				ID:          user.ID,
				IsSuper:     user.IsSuper(),
				Permissions: user.Permissions(),
			})
		}

		remoteAddr, err := realip.FromRequest(r, h.Proxies...)
		if err != nil {
			remoteAddr = r.RemoteAddr

			h.Log.Error("handler middleware: realip from request", "error", err)
		}

		logger := h.Log.With(
			"id", requestID,
			"method", r.Method,
			"remoteAddr", remoteAddr,
			"url", r.URL.String(),
			"user", userID,
		)

		ctx = context.WithValue(ctx, ctxLogger, logger)
		ctx = context.WithValue(ctx, ctxConfig, config)
		ctx = context.WithValue(ctx, ctxUser, user)
		ctx = context.WithValue(ctx, ctxPassport, passport)
		r = r.WithContext(ctx)

		systemConfigPath := h.mux.Path(h.systemConfigPathName)
		if r.Method == http.MethodGet && config.RequiresSetup && r.URL.Path != systemConfigPath && filepath.Ext(r.URL.Path) == "" {
			http.Redirect(w, r, systemConfigPath, http.StatusSeeOther)

			return
		}

		// The redirect key in the session is supposed to be a one-time temporary
		// redirect target, so we ensure it's deleted if we're visiting the target
		if h.Sessions.GetString(ctx, sess.Redirect) == r.URL.String() {
			h.Sessions.Delete(ctx, sess.Redirect)
		}

		next(w, r)
	}
}

func (h *Handler) RenewSession(ctx context.Context) ([]byte, error) {
	if err := csrf.RenewToken(ctx); err != nil {
		return nil, fmt.Errorf("renew CSRF token: %w", err)
	}

	if err := h.Sessions.Renew(ctx); err != nil {
		return nil, err
	}

	return csrf.MaskedToken(ctx), nil
}

func (h *Handler) Logger(ctx context.Context) *slog.Logger {
	value := ctx.Value(ctxLogger)
	if value == nil {
		return slog.Default()
	}

	logger, ok := value.(*slog.Logger)
	if !ok {
		panic(fmt.Sprintf("could not assert logger as %T", logger))
	}

	return logger
}

func (h *Handler) Config(ctx context.Context) *system.Config {
	value := ctx.Value(ctxConfig)
	if value == nil {
		return &system.Config{}
	}

	config, ok := value.(*system.Config)
	if !ok {
		panic(fmt.Sprintf("could not assert config as %T", config))
	}

	return config
}

func (h *Handler) User(ctx context.Context) *account.User {
	value := ctx.Value(ctxUser)
	if value == nil {
		return &account.User{}
	}

	user, ok := value.(*account.User)
	if !ok {
		panic(fmt.Sprintf("could not assert user as %T", user))
	}

	return user
}

func (h *Handler) Passport(ctx context.Context) guard.Passport {
	value := ctx.Value(ctxPassport)
	if value == nil {
		return guard.Passport{}
	}

	passport, ok := value.(guard.Passport)
	if !ok {
		panic(fmt.Sprintf("could not assert system passport as %T", passport))
	}

	return passport
}

func (h *Handler) PassportByEmail(ctx context.Context, email string) (guard.Passport, error) {
	user, err := h.Repo.Account.FindUserByEmail(ctx, email)
	if err != nil {
		return guard.Passport{}, fmt.Errorf("find user by email: %w", err)
	}

	config := h.Config(ctx)

	p := guard.NewPassport(config.RequiresSetup, guard.User{
		ID:          user.ID,
		IsSuper:     user.IsSuper(),
		Permissions: user.Permissions(),
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
		tmpl = errsx.Must(tmpl.ParseFS(h.files, pattern))
	}

	h.templates[name] = tmpl

	return tmpl
}

func (h *Handler) email(name string) *template.Template {
	return h.template(name, "email/"+name+".tmpl")
}

func (h *Handler) emailContentFunc(name string, dataFunc emailDataFunc) (emailContent, error) {
	var content emailContent

	data := emailData{
		URL: URL{
			Scheme:   h.Tenant.Scheme,
			Host:     h.Tenant.Host,
			Hostname: h.Tenant.Hostname,
			Port:     h.Tenant.Port,
		},
	}

	if dataFunc != nil {
		dataFunc(&data)
	}

	email := h.email(name)

	var buf bytes.Buffer

	for _, name := range []string{"subject", "plain", "html"} {
		tmpl := email.Lookup(name)
		if tmpl == nil {
			continue
		}

		buf.Reset()

		if err := tmpl.Execute(&buf, data); err != nil {
			return content, fmt.Errorf("execute email template: %w", err)
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

func (h *Handler) emailContent(view string, vars Vars) (emailContent, error) {
	return h.emailContentFunc(view, func(data *emailData) {
		data.Vars = data.Vars.Merge(vars)
	})
}

func (h *Handler) SendEmail(ctx context.Context, recipients EmailRecipients, view string, vars Vars) error {
	content, err := h.emailContent(view, vars)
	if err != nil {
		return fmt.Errorf("email content: %w", err)
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

	return h.Tenant.Email.Mailer.Send(ctx, msg)
}

func (h *Handler) SendSMS(ctx context.Context, to, body string) error {
	config, err := h.Repo.System.FindConfig(ctx)
	if err != nil {
		return fmt.Errorf("find config: %w", err)
	}

	// TODO: Reuse client for as long as Twilio config hasn't changed
	messager := sms.NewTwilioClient(&httpClient, config.TwilioSID, config.TwilioToken)

	return messager.Send(ctx, config.TwilioFromTel, to, body)
}

func (h *Handler) SendTOTPSMS(email, tel string) error {
	ctx := context.Background()

	user, err := h.Repo.Account.FindUserByEmail(ctx, email)
	if err != nil {
		return fmt.Errorf("find user by email: %w", err)
	}

	totp, err := user.GenerateTOTP()
	if err != nil {
		return fmt.Errorf("generate TOTP: %w", err)
	}

	if tel == "" {
		tel = user.TOTPTel
	}

	return h.SendSMS(ctx, tel, totp)
}

func (h *Handler) view(name string) *template.Template {
	return h.template(name, "partial/*.tmpl", "view/"+name+".tmpl", "master.tmpl")
}

func (h *Handler) ViewFunc(w http.ResponseWriter, r *http.Request, status int, view string, dataFunc ViewDataFunc) {
	ctx := r.Context()
	config := h.Config(ctx)
	passport := h.Passport(ctx)

	data := ViewData{
		View:   view,
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
			KnownPasswordBreachCount: h.Sessions.GetInt(ctx, sess.KnownPasswordBreachCount),
		},
		Config:   config,
		Passport: passport,
	}

	if vars, ok := h.viewVarsFuncs[view]; ok {
		defaults, err := vars(r)
		if err != nil {
			h.ErrorView(w, r, "vars", err, "error", nil)

			return
		}

		data.Vars = data.Vars.Merge(defaults)
	}

	if dataFunc != nil {
		dataFunc(&data)
	}

	// Make sure the current view name isn't overwritten by a user function
	data.View = view

	var buf bytes.Buffer

	if err := h.view(view).ExecuteTemplate(&buf, "master", data); err != nil {
		h.Logger(ctx).Error("execute view template", "error", err)

		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)

		return
	}

	w.Header().Set("content-type", "text/html")
	w.WriteHeader(status)

	if _, err := buf.WriteTo(w); err != nil {
		h.Logger(ctx).Error("write view template response", "error", err)
	}
}

func (h *Handler) View(w http.ResponseWriter, r *http.Request, status int, view string, vars Vars) {
	h.ViewFunc(w, r, status, view, func(data *ViewData) {
		data.Vars = data.Vars.Merge(vars)
	})
}

func (h *Handler) ErrorViewFunc(w http.ResponseWriter, r *http.Request, msg string, err error, view string, dataFunc ViewDataFunc) {
	ctx := r.Context()

	h.Logger(ctx).Error(msg, "error", err)

	status := httputil.ErrorStatus(err)

	h.ViewFunc(w, r, status, view, func(data *ViewData) {
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

			var errs errsx.Map
			if errors.As(err, &errs) {
				data.Errors = errs
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
}

func (h *Handler) ErrorView(w http.ResponseWriter, r *http.Request, msg string, err error, view string, vars Vars) {
	h.ErrorViewFunc(w, r, msg, err, view, func(data *ViewData) {
		data.Vars = data.Vars.Merge(vars)
	})
}

func (h *Handler) ErrorJSON(w http.ResponseWriter, r *http.Request, msg string, err error) {
	ctx := r.Context()

	h.Logger(ctx).Error(msg, "error", err)

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

		var errs errsx.Map
		if errors.As(err, &errs) {
			detail["fields"] = errs
		}
	}

	if err := json.NewEncoder(w).Encode(detail); err != nil {
		h.Logger(ctx).Error("write JSON response", "error", err)
	}
}

func (h *Handler) JSON(w http.ResponseWriter, r *http.Request, data any) bool {
	w.Header().Set("content-type", "application/json")

	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.ErrorJSON(w, r, "encode JSON", err)

		return false
	}

	return true
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

type PredicateFunc func(p guard.Passport) bool

func (h *Handler) RequireSignIn(w http.ResponseWriter, r *http.Request) bool {
	ctx := r.Context()
	isSignedIn := h.Sessions.GetBool(ctx, sess.IsSignedIn)

	if !isSignedIn {
		h.Sessions.Set(ctx, sess.Redirect, r.URL.String())

		http.Redirect(w, r, h.mux.Path(h.signInPathName), http.StatusSeeOther)

		return false
	}

	return true
}

func (h *Handler) RequireSignInIf(check PredicateFunc) router.BeforeHookFunc {
	return func(w http.ResponseWriter, r *http.Request) bool {
		ctx := r.Context()
		isSignedIn := h.Sessions.GetBool(ctx, sess.IsSignedIn)
		passport := h.Passport(ctx)

		if !isSignedIn && check(passport) {
			h.Sessions.Set(ctx, sess.Redirect, r.URL.String())

			http.Redirect(w, r, h.mux.Path(h.signInPathName), http.StatusSeeOther)

			return false
		}

		return true
	}
}

func (h *Handler) RequireAuth(check PredicateFunc) router.BeforeHookFunc {
	return func(w http.ResponseWriter, r *http.Request) bool {
		ctx := r.Context()
		passport := h.Passport(ctx)

		if !check(passport) {
			h.ErrorView(w, r, "require auth", app.ErrUnauthorised, "error", nil)

			return false
		}

		return true
	}
}
