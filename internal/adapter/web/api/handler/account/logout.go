package account

import (
	"encoding/base64"
	"net/http"

	"github.com/polyscone/tofu/internal/adapter/web/handler"
	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/http/router"
)

func Logout(svc *handler.Services, mux *router.ServeMux) {
	mux.Post("/logout", logoutPost(svc))
}

func logoutPost(svc *handler.Services) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		csrfToken, err := svc.RenewSession(ctx)
		if svc.ErrorJSON(w, r, errors.Tracef(err)) {
			return
		}

		svc.Sessions.Destroy(r.Context())

		svc.JSON(w, r, map[string]any{
			"csrfToken": base64.RawURLEncoding.EncodeToString(csrfToken),
		})
	}
}
