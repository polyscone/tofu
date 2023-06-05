package admin

import (
	"net/http"

	"github.com/polyscone/tofu/internal/adapter/web/ui/handler"
	"github.com/polyscone/tofu/internal/pkg/http/router"
)

func Dashboard(h *handler.Handler, guard *handler.Guard, mux *router.ServeMux) {
	mux.Get("/", dashboardGet(h), "admin.dashboard")
}

func dashboardGet(h *handler.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h.View(w, r, http.StatusOK, "admin/dashboard", nil)
	}
}
