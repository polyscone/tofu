package account

import (
	"net/http"

	"github.com/polyscone/tofu/internal/pkg/http/middleware"
	"github.com/polyscone/tofu/internal/pkg/http/router"
	"github.com/polyscone/tofu/internal/web/api"
	"github.com/polyscone/tofu/internal/web/auth"
	"github.com/polyscone/tofu/internal/web/httputil"
)

func RegisterSignUpHandlers(h *api.Handler, mux *router.ServeMux) {
	mux.HandleFunc("POST /account/sign-up", signUpPost(h))
}

func signUpPost(h *api.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Email string
		}
		if err := httputil.DecodeRequestJSON(&input, r); err != nil {
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
