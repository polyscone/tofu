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

	"github.com/polyscone/tofu/app"
	"github.com/polyscone/tofu/app/account"
	"github.com/polyscone/tofu/app/system"
	"github.com/polyscone/tofu/pkg/cache"
	"github.com/polyscone/tofu/pkg/csrf"
	"github.com/polyscone/tofu/pkg/errsx"
	"github.com/polyscone/tofu/pkg/realip"
	"github.com/polyscone/tofu/pkg/session"
	"github.com/polyscone/tofu/pkg/sms"
	"github.com/polyscone/tofu/pkg/smtp"
	"github.com/polyscone/tofu/pkg/uuid"
	"github.com/polyscone/tofu/web/guard"
	"github.com/polyscone/tofu/web/sess"
)

var httpClient = http.Client{Timeout: 10 * time.Second}

type ctxKey int

const (
	ctxLogger ctxKey = iota
	ctxConfig
	ctxUser
	ctxPassport
)

type Handler struct {
	*Tenant

	templates *cache.Cache[string, *template.Template]
	Sessions  *session.Manager
}

func New(tenant *Tenant) *Handler {
	return &Handler{
		Tenant:    tenant,
		templates: cache.New[string, *template.Template](),
		Sessions:  session.NewManager(tenant.Repo.Web),
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
			logger.Error("handler middleware: new v4 UUID", "error", err)

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
		userID := h.Sessions.GetString(ctx, sess.UserID)
		isSignedIn := h.Sessions.GetBool(ctx, sess.IsSignedIn)
		isAwaitingTOTP := h.Sessions.GetBool(ctx, sess.IsAwaitingTOTP)
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
		if !h.Sessions.GetBool(ctx, sess.IsSignedIn) {
			passport = h.PassportByUser(ctx, nil)
		} else {
			passport = h.PassportByUser(ctx, user)
		}

		// Only set the user id in the logger if user isn't in the context yet
		if ok := ctx.Value(ctxUser) != nil; !ok {
			logger = logger.With("user", userID)

			ctx = context.WithValue(ctx, ctxLogger, logger)
		}

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

func (h *Handler) RenewSession(ctx context.Context) ([]byte, error) {
	if err := csrf.RenewToken(ctx); err != nil {
		return nil, fmt.Errorf("renew CSRF token: %w", err)
	}

	if err := h.Sessions.Renew(ctx); err != nil {
		return nil, err
	}

	return csrf.MaskedToken(ctx), nil
}

func (h *Handler) template(files fs.FS, patterns []string, funcs template.FuncMap, name string) *template.Template {
	tmpl := template.New(name).Option("missingkey=default").Funcs(funcs)

	for _, pattern := range patterns {
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

func (h *Handler) Template(files fs.FS, patterns []string, funcs template.FuncMap, name string) *template.Template {
	if h.Tenant.Dev {
		return h.template(files, patterns, funcs, name)
	}

	return h.templates.LoadOrStore(name, func() *template.Template {
		return h.template(files, patterns, funcs, name)
	})
}

func (h *Handler) SendEmail(ctx context.Context, templateFiles fs.FS, templatePaths []string, funcs template.FuncMap, from, to, view string, vars Vars) error {
	data := struct {
		URL  URL
		App  AppData
		Vars Vars
	}{
		URL: URL{
			Scheme: h.Scheme,
			Host:   h.Host,
		},
		App: AppData{
			Name:        app.Name,
			ShortName:   app.ShortName,
			Description: app.Description,
			ThemeColour: app.ThemeColour,
		},
		Vars: vars,
	}

	var buf bytes.Buffer
	var subject, plain, html string
	email := h.Template(templateFiles, templatePaths, funcs, view)
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

	return h.Email.Send(ctx, msg)
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
