package page

import (
	"net/http"

	"github.com/polyscone/tofu/internal/httpx/router"
	"github.com/polyscone/tofu/web/site/ui"
)

func RegisterHandlers(h *ui.Handler, mux *router.ServeMux) {
	mux.HandleFunc("GET /{$}", homeGet(h), "page.home")
}

func homeGet(h *ui.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h.HTML.View(w, r, http.StatusOK, "page/home", nil)
	}
}
