package page

import (
	"net/http"

	"github.com/polyscone/tofu/internal/adapter/web/ui/handler"
	"github.com/polyscone/tofu/internal/pkg/http/router"
)

func Home(h *handler.Handler, mux *router.ServeMux) {
	mux.Get("/", homeGet(h), "page.home")
}

func homeGet(h *handler.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h.View(w, r, http.StatusOK, "page/home", nil)
	}
}
