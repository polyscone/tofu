package account

import (
	"net/http"

	"github.com/polyscone/tofu/internal/adapter/web/ui/handler"
	"github.com/polyscone/tofu/internal/pkg/http/router"
)

func Dashboard(h *handler.Handler, guard *handler.Guard, mux *router.ServeMux) {
	mux.Get("/", dashboardGet(h), "account.dashboard")

	guard.RequireSignIn(mux.Path("account.dashboard"))
}

func dashboardGet(h *handler.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h.View(w, r, http.StatusOK, "account/dashboard", nil)
	}
}
