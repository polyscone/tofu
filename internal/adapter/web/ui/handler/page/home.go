package page

import (
	"net/http"

	"github.com/polyscone/tofu/internal/adapter/web/ui/handler"
	"github.com/polyscone/tofu/internal/pkg/http/router"
)

func Home(svc *handler.Services, mux *router.ServeMux) {
	mux.Get("/", homeGet(svc), "page/home")
}

func homeGet(svc *handler.Services) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		svc.Render(w, r, http.StatusOK, "page/home", nil)
	}
}
