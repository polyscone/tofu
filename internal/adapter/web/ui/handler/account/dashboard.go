package account

import (
	"net/http"

	"github.com/polyscone/tofu/internal/adapter/web/ui/handler"
	"github.com/polyscone/tofu/internal/pkg/http/router"
)

func Dashboard(svc *handler.Services, mux *router.ServeMux) {
	mux.Get("/dashboard", dashboardGet(svc), "account.dashboard")
}

func dashboardGet(svc *handler.Services) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		svc.Render(w, r, http.StatusOK, "account/dashboard", nil)
	}
}