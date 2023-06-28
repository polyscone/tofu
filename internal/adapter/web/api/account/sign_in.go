package account

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/polyscone/tofu/internal/adapter/web/api"
	"github.com/polyscone/tofu/internal/adapter/web/auth"
	"github.com/polyscone/tofu/internal/adapter/web/httputil"
	"github.com/polyscone/tofu/internal/app"
	"github.com/polyscone/tofu/internal/app/account"
	"github.com/polyscone/tofu/internal/pkg/http/middleware"
	"github.com/polyscone/tofu/internal/pkg/http/router"
)

func SignIn(h *api.Handler, mux *router.ServeMux) {
	mux.Prefix("/sign-in", func(mux *router.ServeMux) {
		mux.Post("/", signInPost(h))
	})
}

func signInPost(h *api.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Email    string
			Password string
		}
		if err := httputil.DecodeJSON(&input, r.Body); err != nil {
			h.ErrorJSON(w, r, "decode JSON", err)

			return
		}

		ctx := r.Context()

		if err := auth.SignInWithPassword(ctx, h.Handler, w, r, input.Email, input.Password); err != nil {
			if !errors.Is(err, account.ErrSignInThrottled) {
				err = fmt.Errorf("%w: %w", app.ErrBadRequest, err)
			}

			h.ErrorJSON(w, r, "sign in with password", err)

			return
		}

		w.Header().Set(middleware.CSRFTokenHeaderName, httputil.MaskedCSRFToken(ctx))

		h.JSON(w, r, map[string]any{
			"email": input.Email,
		})
	}
}
