package account

import (
	"net/http"

	"github.com/polyscone/tofu/internal/adapter/web/api"
	"github.com/polyscone/tofu/internal/adapter/web/auth"
	"github.com/polyscone/tofu/internal/adapter/web/httputil"
	"github.com/polyscone/tofu/internal/app/account"
	"github.com/polyscone/tofu/internal/pkg/http/middleware"
	"github.com/polyscone/tofu/internal/pkg/http/router"
)

func verifyRoutes(h *api.Handler, mux *router.ServeMux) {
	mux.HandleFunc("POST /account/verify", verifyPost(h))
}

func verifyPost(h *api.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Token         string
			Password      string
			PasswordCheck string
		}
		if err := httputil.DecodeJSON(&input, r.Body); err != nil {
			h.ErrorJSON(w, r, "decode JSON", err)

			return
		}

		ctx := r.Context()

		email, behaviour, err := auth.Verify(ctx, h.Handler, w, r, input.Token, input.Password, input.PasswordCheck)
		if err != nil {
			h.ErrorJSON(w, r, "verify sign up", err)

			return
		}

		w.Header().Set(middleware.CSRFTokenHeaderName, httputil.MaskedCSRFToken(ctx))

		h.JSON(w, r, http.StatusOK, map[string]any{
			"email":       email,
			"isActivated": behaviour == account.VerifyUserActivate,
		})
	}
}
