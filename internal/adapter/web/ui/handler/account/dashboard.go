package account

import (
	"net/http"

	"github.com/polyscone/tofu/internal/adapter/web/handler"
	"github.com/polyscone/tofu/internal/pkg/http/router"
)

func Dashboard(svc *handler.Services, mux *router.ServeMux, guard *handler.Guard) {
	mux.Get("/", dashboardGet(svc), "account.dashboard")

	guard.RequireSignIn(mux.Path("account.dashboard"))
}

func dashboardGet(svc *handler.Services) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		svc.View(w, r, http.StatusOK, "account/dashboard", nil)
	}
}
