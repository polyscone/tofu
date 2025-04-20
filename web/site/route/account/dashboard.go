package account

import (
	"net/http"

	"github.com/polyscone/tofu/internal/httpx/router"
	"github.com/polyscone/tofu/web/site/ui"
)

func RegisterDashboardHandlers(h *ui.Handler, mux *router.ServeMux) {
	mux.HandleFunc("GET /account", dashboardGet(h), "account.dashboard")
}

func dashboardGet(h *ui.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if h.RequireSignIn(w, r) {
			return
		}

		h.HTML.View(w, r, http.StatusOK, "account/dashboard", nil)
	}
}
