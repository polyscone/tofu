package account

import (
	"net/http"

	"github.com/polyscone/tofu/internal/adapter/web/api"
	"github.com/polyscone/tofu/internal/adapter/web/auth"
	"github.com/polyscone/tofu/internal/adapter/web/httputil"
	"github.com/polyscone/tofu/internal/pkg/http/middleware"
	"github.com/polyscone/tofu/internal/pkg/http/router"
)

func signUpRoutes(h *api.Handler, mux *router.ServeMux) {
	mux.Prefix("/sign-up", func(mux *router.ServeMux) {
		mux.Post("/", signUpPost(h))
	})
}

func signUpPost(h *api.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Email string
		}
		if err := httputil.DecodeJSON(&input, r.Body); err != nil {
			h.ErrorJSON(w, r, "decode JSON", err)

			return
		}

		ctx := r.Context()

		if err := auth.SignUp(ctx, h.Handler, w, r, input.Email); err != nil {
			h.ErrorJSON(w, r, "sign up", err)

			return
		}

		w.Header().Set(middleware.CSRFTokenHeaderName, httputil.MaskedCSRFToken(ctx))

		h.JSON(w, r, http.StatusOK, SessionData(ctx, h))
	}
}