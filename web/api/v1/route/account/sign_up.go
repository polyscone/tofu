package account

import (
	"net/http"

	"github.com/polyscone/tofu/internal/httpx"
	"github.com/polyscone/tofu/internal/httpx/middleware"
	"github.com/polyscone/tofu/internal/httpx/router"
	"github.com/polyscone/tofu/web/api/v1/ui"
	"github.com/polyscone/tofu/web/auth"
)

func RegisterSignUpHandlers(h *ui.Handler, mux *router.ServeMux) {
	mux.HandleFunc("POST /api/v1/account/sign-up", signUpPost(h))
}

func signUpPost(h *ui.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Email string
		}
		if err := httpx.DecodeRequestJSON(&input, r); err != nil {
			h.ErrorJSON(w, r, "decode JSON", err)

			return
		}

		ctx := r.Context()

		if err := auth.SignUp(ctx, h.Handler, w, r, input.Email); err != nil {
			h.ErrorJSON(w, r, "sign up", err)

			return
		}

		w.Header().Set(middleware.CSRFTokenHeaderName, httpx.MaskedCSRFToken(ctx))

		h.JSON(w, r, http.StatusOK, SessionData(ctx, h))
	}
}
