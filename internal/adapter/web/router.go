package web

import (
	"embed"
	"io/fs"
	"net/http"

	"github.com/polyscone/tofu/internal/adapter/web/handler"
	"github.com/polyscone/tofu/internal/pkg/dev"
	"github.com/polyscone/tofu/internal/pkg/errsx"
	"github.com/polyscone/tofu/internal/pkg/fstack"
)

//go:embed "all:ui/public"
var files embed.FS

const publicDir = "ui/public"

var publicFiles = fstack.New(dev.RelDirFS(publicDir), errsx.Must(fs.Sub(files, publicDir)))

func NewRouter(tenant *handler.Tenant) http.Handler {
	mux := http.NewServeMux()
	h := handler.New(tenant)

	// API routes should always be setup before site/PWA routes because routes
	// for the site and PWA could include fallback routes for serving files
	// which would prevent API routes from being run if they were setup after
	// when the router finds a match in the order routes are declared
	mux.Handle("/api/v1/", http.StripPrefix("/api/v1", NewAPIRouter(h)))

	switch tenant.Kind {
	case "site":
		mux.Handle("/", NewSiteRouter(h))

	case "pwa":
		mux.Handle("/", NewPWARouter(h))
	}

	return mux
}
