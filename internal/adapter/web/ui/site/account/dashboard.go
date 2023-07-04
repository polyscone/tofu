package account

import (
	"github.com/polyscone/tofu/internal/adapter/web/ui"
	"github.com/polyscone/tofu/internal/pkg/http/router"
)

func Dashboard(h *ui.Handler, mux *router.ServeMux) {
	mux.Get("/", h.HTML.Handler("site/account/dashboard"), "account.dashboard")

	mux.Before(h.RequireSignIn, mux.Path("account.dashboard"))
}
