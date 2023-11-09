package account

import (
	"github.com/polyscone/tofu/internal/adapter/web/ui"
	"github.com/polyscone/tofu/internal/pkg/http/router"
)

func dashboardRoutes(h *ui.Handler, mux *router.ServeMux) {
	mux.Prefix("/", func(mux *router.ServeMux) {
		mux.Before(h.RequireSignIn)

		mux.Get("/", h.HTML.Handler("site/account/dashboard"), "account.dashboard")
	})
}
