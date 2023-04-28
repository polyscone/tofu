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

	// Redirects
	mux.Redirect(http.MethodGet, "/favicon.ico", "/favicon.png", http.StatusTemporaryRedirect)

	// Pages
	mux.Get("/", page.HomeGet(svc), "page/home")

	// Account
	mux.Prefix("/account", func(mux *router.ServeMux) {
		mux.Get("/dashboard", account.DashboardGet(svc), "account/dashboard")

		mux.Get("/activate", account.ActivateGet(svc), "account/activate")
		mux.Post("/activate", account.ActivatePost(svc, ui.tokens), "account/activate.post")

		mux.Get("/register", account.RegisterGet(svc), "account/register")
		mux.Post("/register", account.RegisterPost(svc), "account/register.post")

		mux.Get("/login", account.LoginGet(svc), "account/login")
		mux.Post("/login", account.LoginPost(svc), "account/login.post")

		mux.Post("/logout", account.LogoutPost(svc), "account/logout.post")

		mux.Get("/forgotten-password", account.ForgottenPasswordGet(svc), "account/forgotten_password")
		mux.Post("/forgotten-password", account.ForgottenPasswordPost(svc, ui.tokens), "account/forgotten_password.post")
		mux.Put("/forgotten-password", account.ForgottenPasswordPut(svc, ui.tokens), "account/forgotten_password.put")

		mux.Get("/change-password", account.ChangePasswordGet(svc), "account/change_password")
		mux.Put("/change-password", account.ChangePasswordPut(svc), "account/change_password.put")

		mux.Get("/totp", account.TOTPGet(svc), "account/totp")
		mux.Post("/totp/app", account.TOTPSetupAppPost(svc), "account/totp/app.post")
		mux.Post("/totp/verify", account.TOTPVerifyPost(svc), "account/totp/verify.post")
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
