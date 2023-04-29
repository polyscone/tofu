package ui

import (
	"embed"
	"html/template"
	"io/fs"
	"net/http"

	"github.com/polyscone/tofu/internal/adapter/web/httputil"
	"github.com/polyscone/tofu/internal/adapter/web/smtp"
	"github.com/polyscone/tofu/internal/adapter/web/token"
	"github.com/polyscone/tofu/internal/adapter/web/ui/handler"
	"github.com/polyscone/tofu/internal/adapter/web/ui/handler/account"
	"github.com/polyscone/tofu/internal/adapter/web/ui/handler/page"
	"github.com/polyscone/tofu/internal/pkg/command"
	"github.com/polyscone/tofu/internal/pkg/dev"
	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/fstack"
	"github.com/polyscone/tofu/internal/pkg/http/router"
	"github.com/polyscone/tofu/internal/pkg/session"
)

//go:embed "public" "template"
var files embed.FS

type Options struct {
	Dev bool
}

type UI struct {
	svc    *handler.Services
	mux    *router.ServeMux
	tokens token.Repo
}

func New(bus command.Bus, sessions *session.Manager, tokens token.Repo, mailer smtp.Mailer, opts Options) *UI {
	files := fs.FS(files)
	templateFiles := fstack.New(dev.RelDirFS("template"), errors.Must(fs.Sub(files, "template")))
	mux := router.NewServeMux()
	svc := handler.NewServices(mux, bus, sessions, mailer, handler.Options{
		Cache: !opts.Dev,
		Files: templateFiles,
		Funcs: template.FuncMap{
			"StatusText": http.StatusText,
			"Path":       mux.Path,
		},
	})

	return &UI{
		svc:    svc,
		mux:    mux,
		tokens: tokens,
	}
}

func (ui *UI) Routes() http.Handler {
	svc, mux := ui.svc, ui.mux

	// Rewrites
	mux.Rewrite(http.MethodGet, "/favicon.ico", "/favicon.png")

	// Pages
	page.Home(svc, mux)

	// Account
	mux.Prefix("/account", func(mux *router.ServeMux) {
		account.Dashboard(svc, mux)
		account.Activate(svc, mux, ui.tokens)
		account.Register(svc, mux)
		account.Login(svc, mux)
		account.Logout(svc, mux)
		account.ForgottenPassword(svc, mux, ui.tokens)
		account.ChangePassword(svc, mux)
		account.TOTP(svc, mux)
	})

	// Public static file handler
	publicFiles := fstack.New(dev.RelDirFS("public"), errors.Must(fs.Sub(files, "public")))
	mux.GetHandler("/:rest", http.FileServer(http.FS(publicFiles)))

	// Generic not found handler
	mux.NotFound(func(w http.ResponseWriter, r *http.Request) {
		svc.RenderError(w, r, errors.Tracef("%w: %v %v", httputil.ErrNotFound, r.Method, r.URL), "error", nil)
	})

	// Generic method not allowed handler
	mux.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
		svc.RenderError(w, r, errors.Tracef("%w: %v %v", httputil.ErrMethodNotAllowed, r.Method, r.URL), "error", nil)
	})

	return ui.mux
}

func (ui *UI) ErrorHandler(w http.ResponseWriter, r *http.Request, err error) {
	ui.svc.RenderError(w, r, errors.Tracef(err), "error", nil)
}
