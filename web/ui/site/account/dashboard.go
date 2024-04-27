package account

import (
	"github.com/polyscone/tofu/pkg/http/router"
	"github.com/polyscone/tofu/web/ui"
)

func RegisterDashboardHandlers(h *ui.Handler, mux *router.ServeMux) {
	mux.Group(func(mux *router.ServeMux) {
		mux.Before(h.RequireSignIn)

		mux.HandleFunc("GET /account", h.HTML.HandlerFunc("site/account/dashboard"), "account.dashboard")
	})
}
