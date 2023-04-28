package web

import (
	"bytes"
	"context"
	"embed"
	"html/template"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/polyscone/tofu/internal/adapter/web/api"
	"github.com/polyscone/tofu/internal/adapter/web/httputil"
	"github.com/polyscone/tofu/internal/adapter/web/smtp"
	"github.com/polyscone/tofu/internal/adapter/web/token"
	"github.com/polyscone/tofu/internal/adapter/web/ui"
	"github.com/polyscone/tofu/internal/pkg/background"
	"github.com/polyscone/tofu/internal/pkg/command"
	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/event"
	"github.com/polyscone/tofu/internal/pkg/fstack"
	"github.com/polyscone/tofu/internal/pkg/http/middleware"
	"github.com/polyscone/tofu/internal/pkg/http/router"
	"github.com/polyscone/tofu/internal/pkg/logger"
	"github.com/polyscone/tofu/internal/pkg/session"
	"github.com/polyscone/tofu/internal/pkg/size"
	"github.com/polyscone/tofu/internal/pkg/valobj/text"
	"github.com/polyscone/tofu/internal/port/account"
)

//go:embed "files/email"
var embeddedFiles embed.FS

var tmplFuncs = template.FuncMap{}

type MsgContent struct {
	Subject string
	Plain   string
	HTML    string
}

type Options struct {
	Dev      bool
	Insecure bool
	Proxies  []string
}

type Handler struct {
	dev         bool
	mux         *router.ServeMux
	files       fs.FS
	templatesMu sync.RWMutex
	templates   map[string]*template.Template
}

func NewHandler(bus command.Bus, broker event.Broker, sessions session.Repo, tokens token.Repo, mailer smtp.Mailer, opts Options) http.Handler {
	files := fs.FS(embeddedFiles)

	dir := "internal/adapter/web"
	if info, err := os.Stat(dir); err == nil && info.IsDir() {
		files = fstack.New(os.DirFS(dir), files)
	}

	templates := make(map[string]*template.Template)

	sm := session.NewManager(sessions)
	api := api.New(bus, sm, tokens, mailer)
	ui := ui.New(bus, sm, tokens, mailer, ui.Options{
		Dev: opts.Dev,
	})

	errorHandler := func(w http.ResponseWriter, r *http.Request, err error) {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			api.ErrorHandler(w, r, err)
		} else {
			ui.ErrorHandler(w, r, err)
		}
	}

	mux := router.NewServeMux()

	mux.Use(middleware.Recover(errorHandler))
	mux.Use(middleware.MethodOverride)
	mux.Use(middleware.RateLimit(50, 1, &middleware.RateLimitConfig{
		ErrorHandler:   errorHandler,
		TrustedProxies: opts.Proxies,
	}))
	mux.Use(middleware.Session(sm, &middleware.SessionConfig{
		Insecure:     opts.Insecure,
		ErrorHandler: errorHandler,
	}))
	mux.Use(httputil.TraceRequest(sm, errorHandler))
	mux.Use(middleware.NoContent)
	mux.Use(middleware.SecurityHeaders)
	mux.Use(middleware.ETag)
	mux.Use(middleware.CSRF(&middleware.CSRFConfig{
		Insecure:     opts.Insecure,
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

	h := Handler{
		dev:       opts.Dev,
		mux:       mux,
		files:     files,
		templates: templates,
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

			data := struct {
				Token string
			}{
				Token: tok,
			}

			content, err := h.msgContent("activate_account", data)
			if err != nil {
				logger.PrintError(err)

				return
			}

			msg := smtp.Msg{
				From:    "noreply@example.com",
				To:      []string{evt.Email},
				Subject: content.Subject,
				Plain:   content.Plain,
				HTML:    content.HTML,
			}
			if err := mailer.Send(ctx, msg); err != nil {
				logger.PrintError(err)
			}
		})
	})

	return &h
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.mux.ServeHTTP(w, r)
}

func (h *Handler) view(view string) *template.Template {
	h.templatesMu.RLock()

	// Return the cached template only when we're not in a dev environment
	if tmpl := h.templates[view]; tmpl != nil && !h.dev {
		h.templatesMu.RUnlock()

		return tmpl
	}

	h.templatesMu.RUnlock()

	h.templatesMu.Lock()
	defer h.templatesMu.Unlock()

	key := strings.TrimSuffix(filepath.Base(view), ".go.html")

	tmpl := template.New(key).Funcs(tmplFuncs)
	tmpl = errors.Must(tmpl.ParseFS(h.files, "files/email/view/"+view+".go.html"))

	h.templates[key] = tmpl

	return tmpl
}

func (h *Handler) msgContent(view string, data any) (MsgContent, error) {
	var content MsgContent
	var buf bytes.Buffer

	v := h.view(view)

	for _, name := range []string{"subject", "plain", "html"} {
		tmpl := v.Lookup(name)
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
