package web

import (
	"embed"
	"io/fs"
	"net/http"
	"path"
	"strings"

	"github.com/polyscone/tofu/internal/adapter/web/handler"
	"github.com/polyscone/tofu/internal/adapter/web/httputil"
	"github.com/polyscone/tofu/internal/pkg/dev"
	"github.com/polyscone/tofu/internal/pkg/errsx"
	"github.com/polyscone/tofu/internal/pkg/fstack"
	"github.com/polyscone/tofu/internal/pkg/http/router"
)

//go:embed "all:public"
//go:embed "all:template"
var files embed.FS

const (
	publicDir   = "public"
	templateDir = "template"
)

var (
	publicFiles   = fstack.New(dev.RelDirFS(publicDir), errsx.Must(fs.Sub(files, publicDir)))
	templateFiles = fstack.New(dev.RelDirFS(templateDir), errsx.Must(fs.Sub(files, templateDir)))
)

func NewRouter(tenant *handler.Tenant) http.Handler {
	mux := router.NewServeMux()
	h := handler.New(mux, tenant, templateFiles, func() string {
		return mux.Path("account.sign_in")
	})

	tenant.Broker.Listen(accountSignedInWithPasswordHandler(h))
	tenant.Broker.Listen(accountTOTPDisabledHandler(h))
	tenant.Broker.Listen(accountSignedUpHandler(h))

	mux.Redirect(http.MethodGet, "/security.txt", "/.well-known/security.txt", http.StatusMovedPermanently)

	mux.Rewrite(http.MethodGet, "/favicon.ico", "/favicon.png")

	mux.Get("/robots.txt", h.Plain.Handler("file/robots"))
	mux.Get("/.well-known/security.txt", h.Plain.Handler("file/security"))

	switch tenant.Kind {
	case "site":
		setupSiteRoutes(tenant, h, mux)

	case "pwa":
		setupPWARoutes(tenant, h, mux)
	}

	return mux
}

type ErrorHandler func(w http.ResponseWriter, r *http.Request, err error)

func setupPublicFileServerRoute(h *handler.Handler, mux *router.ServeMux, errorHandler ErrorHandler) {
	publicFilesRoot := http.FS(publicFiles)
	fileServer := http.FileServer(publicFilesRoot)
	mux.GetHandler("/:rest*", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upath := r.URL.Path
		if !strings.HasPrefix(upath, "/") {
			upath = "/" + upath
			r.URL.Path = upath
		}
		upath = path.Clean(upath)

		stat, err := fs.Stat(publicFiles, strings.TrimPrefix(upath, "/"))
		if err != nil {
			errorHandler(w, r, err)

			return
		}
		if stat.IsDir() {
			errorHandler(w, r, httputil.ErrForbidden)

			return
		}

		fileServer.ServeHTTP(w, r)
	}))
}
