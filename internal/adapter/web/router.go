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

	switch tenant.Kind {
	case "site":
		mux.Handle("/", NewSiteRouter(h))

	case "pwa":
		mux.Handle("/", NewPWARouter(h))
	}

	mux.Handle("/api/v1/", http.StripPrefix("/api/v1", NewAPIRouter(h)))

	return mux
}
