package admin

import (
	"github.com/polyscone/tofu/http/router"
	"github.com/polyscone/tofu/web/ui"
)

func RegisterDashboardHandlers(h *ui.Handler, mux *router.ServeMux) {
	mux.Group(func(mux *router.ServeMux) {
		mux.Before(h.RequireSignIn)

		mux.HandleFunc("GET /admin", h.HTML.HandlerFunc("site/admin/dashboard"), "admin.dashboard")
	})
}
