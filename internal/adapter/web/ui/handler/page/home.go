package page

import (
	"net/http"

	"github.com/polyscone/tofu/internal/adapter/web/ui/handler"
)

func HomeGet(svc *handler.Services) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		svc.Render(w, r, http.StatusOK, "page/home", nil)
	}
}
