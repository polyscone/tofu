package account

import (
	"net/http"

	"github.com/polyscone/tofu/internal/adapter/web/ui/handler"
)

func DashboardGet(svc *handler.Services) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		svc.Render(w, r, http.StatusOK, "account/dashboard", nil)
	}
}
