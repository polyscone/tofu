package admin

import (
	"github.com/polyscone/tofu/internal/adapter/web/ui/handler"
	"github.com/polyscone/tofu/internal/pkg/http/router"
)

func Dashboard(h *handler.Handler, mux *router.ServeMux) {
	mux.Get("/", h.HandleView("admin/dashboard"), "admin.dashboard")
}
