package web

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/polyscone/tofu/internal/adapter/web/internal/api"
	"github.com/polyscone/tofu/internal/adapter/web/internal/httputil"
	"github.com/polyscone/tofu/internal/adapter/web/internal/smtp"
	"github.com/polyscone/tofu/internal/adapter/web/internal/token"
	"github.com/polyscone/tofu/internal/adapter/web/internal/ui"
	"github.com/polyscone/tofu/internal/pkg/background"
	"github.com/polyscone/tofu/internal/pkg/command"
	"github.com/polyscone/tofu/internal/pkg/event"
	"github.com/polyscone/tofu/internal/pkg/http/middleware"
	"github.com/polyscone/tofu/internal/pkg/http/router"
	"github.com/polyscone/tofu/internal/pkg/logger"
	"github.com/polyscone/tofu/internal/pkg/session"
	"github.com/polyscone/tofu/internal/pkg/size"
	"github.com/polyscone/tofu/internal/pkg/valobj/text"
	"github.com/polyscone/tofu/internal/port/account"
)

type Options struct {
	dev      bool
	insecure bool
	proxies  []string
}

type Option func(opts *Options)

func WithDev(value bool) Option {
	return func(opts *Options) {
		opts.dev = value
	}
}

func WithInsecure(value bool) Option {
	return func(opts *Options) {
		opts.insecure = value
	}
}

func WithProxies(proxies []string) Option {
	return func(opts *Options) {
		opts.proxies = proxies
	}
}

func NewHandler(bus command.Bus, broker event.Broker, sessions session.Repo, tokens token.Repo, mailer smtp.Mailer, _opts ...Option) http.Handler {
	var opts Options
	for _, opt := range _opts {
		opt(&opts)
	}

	broker.Listen(func(evt account.Registered) {
		background.Go(func() {
			ctx := context.Background()

			email, err := text.NewEmail(evt.Email)
			if err != nil {
				logger.PrintError(err)

				return
			}

			tok, err := tokens.AddActivationToken(ctx, email, 48*time.Hour)
			if err != nil {
				logger.PrintError(err)

				return
			}

			msg := smtp.Msg{
				From:    "noreply@example.com",
				To:      []string{evt.Email},
				Subject: "Activate your account",
				Plain:   "Activation code: " + tok,
				HTML:    "<h1>Activation code</h1><p>" + tok + "</p>",
			}
			if err := mailer.Send(ctx, msg); err != nil {
				logger.PrintError(err)
			}
		})
	})

	sm := session.NewManager(sessions)
	api := api.New(bus, sm, tokens, mailer)
	ui := ui.New(bus, sm, ui.WithDev(opts.dev))

	errorHandler := func(w http.ResponseWriter, r *http.Request, err error) {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			api.ErrorHandler(w, r, err)
		} else {
			ui.ErrorHandler(w, r, err)
		}
	}

	mux := router.NewServeMux()

	mux.Use(middleware.Recover(errorHandler))
	mux.Use(middleware.RateLimit(50, 1, &middleware.RateLimitConfig{
		ErrorHandler:   errorHandler,
		TrustedProxies: opts.proxies,
	}))
	mux.Use(middleware.Session(sm, &middleware.SessionConfig{
		Insecure:     opts.insecure,
		ErrorHandler: errorHandler,
	}))
	mux.Use(httputil.TraceRequest(sm, errorHandler))
	mux.Use(middleware.NoContent)
	mux.Use(middleware.SecurityHeaders)
	mux.Use(middleware.ETag)
	mux.Use(middleware.CSRF(&middleware.CSRFConfig{
		Insecure:     opts.insecure,
		ErrorHandler: errorHandler,
	}))
	mux.Use(middleware.Heartbeat("/meta/health"))
	mux.Use(middleware.MaxBytes(func(r *http.Request) int {
		switch r.Method {
		case http.MethodPost, http.MethodPut, http.MethodPatch:
			return 100 * size.Kilobyte
		}

		return 0
	}))

	mux.AnyHandler("/api/v1/:rest", api.Routes())
	mux.AnyHandler("/:rest", ui.Routes())

	return mux
}
