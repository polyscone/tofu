package handler

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"html/template"
	"io/fs"
	"log/slog"
	"net/http"
	"strings"
	"time"
	"unicode"

	"github.com/polyscone/tofu/app"
	"github.com/polyscone/tofu/app/account"
	"github.com/polyscone/tofu/app/system"
	"github.com/polyscone/tofu/internal/cache"
	"github.com/polyscone/tofu/internal/csrf"
	"github.com/polyscone/tofu/internal/errsx"
	"github.com/polyscone/tofu/internal/httpx/realip"
	"github.com/polyscone/tofu/internal/i18n"
	"github.com/polyscone/tofu/internal/session"
	"github.com/polyscone/tofu/internal/smtp"
	"github.com/polyscone/tofu/internal/twilio"
	"github.com/polyscone/tofu/internal/uuid"
	"github.com/polyscone/tofu/web/guard"
)

var httpClient = http.Client{Timeout: 10 * time.Second}

type ctxKey int

const (
	ctxLogger ctxKey = iota
	ctxConfig
	ctxUser
	ctxPassport
	ctxLocale
)

type GuardPredicateFunc func(p guard.Passport) bool
type TemplatePatternsFunc func(view string) []string

type Handler struct {
	*Tenant

	templates *cache.Cache[string, *template.Template]
	Session   Session
}

func New(tenant *Tenant) *Handler {
	return &Handler{
		Tenant:    tenant,
		templates: cache.New[string, *template.Template](),
		Session:   Session{Manager: session.NewManager(tenant.Repo.Web)},
	}
}

func (h *Handler) AttachContextLogger(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// Skip setting up the logger in the context if it already exists
		if ok := ctx.Value(ctxLogger) != nil; ok {
			next(w, r)

			return
		}

		logger := h.Logger(ctx)

		requestID, err := uuid.NewV7()
		if err != nil {
			logger.Error("handler middleware: new v7 UUID", "error", err)

			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)

			return
		}

		remoteAddr, err := realip.FromRequest(r, h.Proxies)
		if err != nil {
			remoteAddr = r.RemoteAddr

			logger.Error("handler middleware: realip from request", "error", err)
		}

		logger = logger.With(
			"id", requestID.String(),
			"method", r.Method,
			"remoteAddr", remoteAddr,
			"url", r.URL.String(),
		)

		ctx = context.WithValue(ctx, ctxLogger, logger)
		r = r.WithContext(ctx)

		next(w, r)
	}
}

func (h *Handler) AttachContext(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		logger := h.Logger(ctx)

		config, err := h.Repo.System.FindConfig(ctx)
		if err != nil {
			logger.Error("handler middleware: find config", "error", err)

			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)

			return
		}

		user := &account.User{}
		userID := h.Session.UserID(ctx)
		isSignedIn := h.Session.IsSignedIn(ctx)
		isAwaitingTOTP := h.Session.IsAwaitingTOTP(ctx)
		if isSignedIn || isAwaitingTOTP {
			u, err := h.Repo.Account.FindUserByID(ctx, userID)
			switch {
			case err == nil:
				user = u

			case !errors.Is(err, app.ErrNotFound):
				logger.Error("handler middleware: find user by id", "error", err)
			}
		}

		var passport guard.Passport
		if h.Session.IsSignedIn(ctx) {
			passport = h.PassportByUser(ctx, user)
		} else {
			passport = h.PassportByUser(ctx, nil)
		}

		// Only set the user id in the logger if user isn't in the context yet
		if ok := ctx.Value(ctxUser) != nil; !ok {
			logger = logger.With("user", userID)

			ctx = context.WithValue(ctx, ctxLogger, logger)
		}

		var candidates []string
		for _, value := range r.Header.Values("accept-language") {
			locales := strings.Split(value, ",")
			for _, locale := range locales {
				locale, _, _ = strings.Cut(locale, ";")
				if strings.ContainsFunc(locale, unicode.IsSpace) {
					locale = strings.TrimSpace(locale)
				}
				if strings.Contains(locale, "_") {
					locale = strings.ReplaceAll(locale, "_", "-")
				}

				candidates = append(candidates, locale)
			}
		}

		locale, _ := i18n.ClosestLocale(candidates)

		ctx = context.WithValue(ctx, ctxConfig, config)
		ctx = context.WithValue(ctx, ctxUser, user)
		ctx = context.WithValue(ctx, ctxPassport, passport)
		ctx = context.WithValue(ctx, ctxLocale, locale)
		r = r.WithContext(ctx)

		// The redirect key in the session is supposed to be a one-time temporary
		// redirect target, so we ensure it's deleted if we're visiting the target
		if h.Session.Redirect(ctx) == r.URL.String() {
			h.Session.DeleteRedirect(ctx)
		}

		next(w, r)
	}
}

func (h *Handler) Logger(ctx context.Context) *slog.Logger {
	value := ctx.Value(ctxLogger)
	if value == nil {
		return h.Tenant.Logger
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

func (h *Handler) PassportByUser(ctx context.Context, user *account.User) guard.Passport {
	return guard.NewPassport(user, h.SuperRole.ID)
}

func (h *Handler) Passport(ctx context.Context) guard.Passport {
	value := ctx.Value(ctxPassport)
	if value == nil {
		return h.PassportByUser(ctx, nil)
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
		return h.PassportByUser(ctx, nil), fmt.Errorf("find user by email: %w", err)
	}

	p := h.PassportByUser(ctx, user)

	return p, nil
}

func (h *Handler) Locale(ctx context.Context) string {
	value := ctx.Value(ctxLocale)
	if value == nil {
		return i18n.FallbackLocale
	}

	locale, ok := value.(string)
	if !ok {
		panic(fmt.Sprintf("could not assert locale as %T", locale))
	}

	return locale
}

func (h *Handler) RenewSession(ctx context.Context) ([]byte, error) {
	if err := csrf.RenewToken(ctx); err != nil {
		return nil, fmt.Errorf("renew CSRF token: %w", err)
	}

	if err := h.Session.Renew(ctx); err != nil {
		return nil, err
	}

	return csrf.MaskedToken(ctx), nil
}

func (h *Handler) T(ctx context.Context, message i18n.Message) string {
	locale := h.Locale(ctx)
	res, err := i18n.T(i18n.DefaultHTMLRuntime, locale, message)
	if err != nil {
		logger := h.Logger(ctx)

		logger.Error("flash errorf i18n T", "err", err)
	}

	return res.AsString().Value
}

func (h *Handler) template(files fs.FS, patterns TemplatePatternsFunc, funcs template.FuncMap, name string) *template.Template {
	tmpl := template.New(name).Option("missingkey=default").Funcs(funcs)

	for _, pattern := range patterns(name) {
		if strings.Contains(pattern, "/com_*") {
			// Ignore errors for com_* because not all folders will have them
			fsTmpl, err := tmpl.ParseFS(files, pattern)
			if err == nil {
				tmpl = fsTmpl
			}
		} else {
			tmpl = errsx.Must(tmpl.ParseFS(files, pattern))
		}
	}

	return tmpl
}

func (h *Handler) Template(files fs.FS, patterns TemplatePatternsFunc, funcs template.FuncMap, name string) *template.Template {
	if h.Tenant.Dev {
		return h.template(files, patterns, funcs, name)
	}

	return h.templates.LoadOrStore(name, func() *template.Template {
		return h.template(files, patterns, funcs, name)
	})
}

func (h *Handler) SendEmail(ctx context.Context, templateFiles fs.FS, templatePatterns TemplatePatternsFunc, funcs template.FuncMap, from, to, view string, vars Vars) error {
	logger := h.Logger(ctx)

	data := ViewData{
		Locale: h.Locale(ctx),
		Now:    time.Now(),
		URL: URL{
			Scheme: h.Scheme,
			Host:   h.Host,
		},
		App: AppData{
			Name:        app.Name,
			ShortName:   app.ShortName,
			Description: app.Description,
			ThemeColour: app.ThemeColour,
			BasePath:    app.BasePath,
		},
		Log:  Logger{logger: logger},
		Vars: vars,
	}

	var buf bytes.Buffer
	var subject, plain, html string
	email := h.Template(templateFiles, templatePatterns, funcs, view)
	for _, view := range []string{"subject", "plain", "html"} {
		tmpl := email.Lookup(view)
		if tmpl == nil {
			continue
		}

		buf.Reset()

		if view == "html" {
			data.I18nRuntime = i18n.DefaultHTMLRuntime
		} else {
			data.I18nRuntime = i18n.DefaultMarkdownRuntime
		}

		if err := tmpl.Execute(&buf, data); err != nil {
			return fmt.Errorf("execute email template: %w", err)
		}

		switch view {
		case "subject":
			subject = buf.String()

		case "plain":
			plain = buf.String()

		case "html":
			html = buf.String()
		}
	}

	msg := smtp.Msg{
		From:    from,
		To:      []string{to},
		Subject: subject,
		Plain:   plain,
		HTML:    html,
	}

	return h.Email.SendEmail(ctx, msg)
}

func (h *Handler) SendSMS(ctx context.Context, to, body string) error {
	config, err := h.Repo.System.FindConfig(ctx)
	if err != nil {
		return fmt.Errorf("find config: %w", err)
	}

	// TODO: Reuse client for as long as Twilio config hasn't changed
	messager := twilio.NewTwilioClient(&httpClient, config.TwilioSID, config.TwilioToken)

	return messager.SendSMS(ctx, config.TwilioFromTel, to, body)
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
