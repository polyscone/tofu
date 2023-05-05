package security

import (
	"encoding/base64"
	"net/http"

	"github.com/polyscone/tofu/internal/adapter/web/handler"
	"github.com/polyscone/tofu/internal/pkg/csrf"
	"github.com/polyscone/tofu/internal/pkg/http/router"
)

func CSRF(svc *handler.Services, mux *router.ServeMux) {
	mux.Get("/csrf", csrfGet(svc))
}

func csrfGet(svc *handler.Services) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		csrfTokenBase64 := base64.RawURLEncoding.EncodeToString(csrf.MaskedToken(ctx))

		svc.JSON(w, r, map[string]any{
			"csrfToken": csrfTokenBase64,
		})
	}
}
