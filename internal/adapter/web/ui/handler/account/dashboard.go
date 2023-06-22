package account

import (
	"github.com/polyscone/tofu/internal/adapter/web/ui/handler"
	"github.com/polyscone/tofu/internal/pkg/http/router"
)

func Dashboard(h *handler.Handler, mux *router.ServeMux) {
	mux.Get("/", h.HandleView("account/dashboard"), "account.dashboard")

	mux.Before(h.RequireSignIn, mux.Path("account.dashboard"))
}
