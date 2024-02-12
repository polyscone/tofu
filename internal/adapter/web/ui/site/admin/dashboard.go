package admin

import (
	"github.com/polyscone/tofu/internal/adapter/web/ui"
	"github.com/polyscone/tofu/internal/pkg/http/router"
)

func RegisterDashboardHandlers(h *ui.Handler, mux *router.ServeMux) {
	mux.Group(func(mux *router.ServeMux) {
		mux.Before(h.RequireSignIn)

		mux.HandleFunc("GET /admin", h.HTML.HandlerFunc("site/admin/dashboard"), "admin.dashboard")
	})
}
