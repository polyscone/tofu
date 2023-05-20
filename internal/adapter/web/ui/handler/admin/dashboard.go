package admin

import (
	"net/http"

	"github.com/polyscone/tofu/internal/adapter/web/handler"
	"github.com/polyscone/tofu/internal/pkg/http/router"
)

func Dashboard(svc *handler.Services, mux *router.ServeMux) {
	mux.Get("/dashboard", dashboardGet(svc), "admin.dashboard")
}

func dashboardGet(svc *handler.Services) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		svc.View(w, r, http.StatusOK, "admin/dashboard", nil)
	}
}
