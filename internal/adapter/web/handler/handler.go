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
	signInPath    func() string
	files         fs.FS
	templatesMu   sync.RWMutex
	templates     map[string]*template.Template
	funcs         template.FuncMap
	viewVarsFuncs map[string]ViewVarsFunc
	mux           *router.ServeMux
	Sessions      *session.Manager
	Plain         *Renderer
	HTML          *Renderer
}

func New(mux *router.ServeMux, tenant *Tenant, files fs.FS, signInPath func() string) *Handler {
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

	h := Handler{
		Tenant:        tenant,
		signInPath:    signInPath,
		files:         files,
		templates:     make(map[string]*template.Template),
		funcs:         funcs,
		viewVarsFuncs: make(map[string]ViewVarsFunc),
		mux:           mux,
		Sessions:      sessions,
	}

	h.Plain = NewRenderer(&h, "text/plain")
	h.HTML = NewRenderer(&h, "text/html")

	return &h
}

func (h *Handler) SetupMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		config, err := h.Repo.System.FindConfig(ctx)
		if err != nil {
			h.Log.Error("handler middleware: find config", "error", err)

			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)

			return
		}

		user := &account.User{}
		userID := h.Sessions.GetInt(ctx, sess.UserID)
		isSignedIn := h.Sessions.GetBool(ctx, sess.IsSignedIn)
		isAwaitingTOTP := h.Sessions.GetBool(ctx, sess.IsAwaitingTOTP)
		if isSignedIn || isAwaitingTOTP {
			u, err := h.Repo.Account.FindUserByID(ctx, userID)
			switch {
			case err == nil:
				user = u

			case err != nil && !errors.Is(err, repository.ErrNotFound):
				h.Log.Error("handler middleware: find user by id", "error", err)
			}
		}

		var passport guard.Passport
		if !isSignedIn {
			passport = guard.NewPassport(config.RequireSetup, guard.User{})
		} else {
			passport = guard.NewPassport(config.RequireSetup, guard.User{
				ID:          user.ID,
				IsSuper:     user.IsSuper(),
				Permissions: user.Permissions(),
			})
		}

		requestID, err := uuid.NewV4()
		if err != nil {
			h.Log.Error("handler middleware: new v4 UUID", "error", err)

			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)

			return
		}

		remoteAddr, err := realip.FromRequest(r, h.Proxies...)
		if err != nil {
			remoteAddr = r.RemoteAddr

			h.Log.Error("handler middleware: realip from request", "error", err)
		}

		logger := h.Log.With(
			"id", requestID.String(),
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

	p := guard.NewPassport(config.RequireSetup, guard.User{
		ID:          user.ID,
		IsSuper:     user.IsSuper(),
		Permissions: user.Permissions(),
	})

	return p, nil
}

func (h *Handler) HasPathPrefix(value string, name string, paramArgPairs ...any) bool {
	p := h.Path(name, paramArgPairs...)
	p = strings.TrimSuffix(p, "/")

	return value == p || strings.HasPrefix(value, p+"/")
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

func (h *Handler) SendEmail(ctx context.Context, recipients EmailRecipients, view string, vars Vars) error {
	var content emailContent

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
		},
		Vars: vars,
	}

	var buf bytes.Buffer
	email := h.template(view, "email/"+view+".tmpl")
	for _, view := range []string{"subject", "plain", "html"} {
		tmpl := email.Lookup(view)
		if tmpl == nil {
			continue
		}

		buf.Reset()

		if err := tmpl.Execute(&buf, data); err != nil {
			return fmt.Errorf("execute email template: %w", err)
		}

		switch view {
		case "subject":
			content.Subject = buf.String()

		case "plain":
			content.Plain = buf.String()

		case "html":
			content.HTML = buf.String()
		}
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
		h.Logger(ctx).Error("write error JSON response", "error", err)
	}
}

func (h *Handler) JSON(w http.ResponseWriter, r *http.Request, data any) {
	ctx := r.Context()

	w.Header().Set("content-type", "application/json")

	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.Logger(ctx).Error("write JSON response", "error", err)
	}
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

		http.Redirect(w, r, h.signInPath(), http.StatusSeeOther)

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

			http.Redirect(w, r, h.signInPath(), http.StatusSeeOther)

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
			h.HTML.ErrorView(w, r, "require auth", app.ErrUnauthorised, "site/error", nil)

			return false
		}

		return true
	}
}
