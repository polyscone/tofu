package system

import (
	"github.com/polyscone/tofu/internal/adapter/web/ui"
	"github.com/polyscone/tofu/internal/pkg/http/router"
)

func Routes(h *ui.Handler, mux *router.ServeMux) {
	mux.Prefix("/system", func(mux *router.ServeMux) {
		setupRoutes(h, mux)
	})
}
