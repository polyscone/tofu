package meta

import (
	"github.com/polyscone/tofu/internal/adapter/web/api"
	"github.com/polyscone/tofu/internal/pkg/http/router"
)

func Routes(h *api.Handler, mux *router.ServeMux) {
	mux.Group("/meta", func(mux *router.ServeMux) {
		healthRoutes(h, mux)
	})
}
