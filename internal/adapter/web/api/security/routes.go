package security

import (
	"github.com/polyscone/tofu/internal/adapter/web/api"
	"github.com/polyscone/tofu/internal/pkg/http/router"
)

func Routes(h *api.Handler, mux *router.ServeMux) {
	mux.Prefix("/security", func(mux *router.ServeMux) {
		csrfRoutes(h, mux)
	})
}
