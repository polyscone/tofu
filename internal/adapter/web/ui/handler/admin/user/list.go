package user

import (
	"net/http"

	"github.com/polyscone/tofu/internal/adapter/web/handler"
	"github.com/polyscone/tofu/internal/pkg/http/router"
)

func List(svc *handler.Services, mux *router.ServeMux) {
	mux.Get("/list", listGet(svc), "admin.user.list")
}

func listGet(svc *handler.Services) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		svc.View(w, r, http.StatusOK, "admin/user/list", nil)
	}
}
