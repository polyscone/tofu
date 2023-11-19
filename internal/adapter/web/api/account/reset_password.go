package account

import (
	"fmt"
	"net/http"

	"github.com/polyscone/tofu/internal/adapter/web/api"
	"github.com/polyscone/tofu/internal/adapter/web/auth"
	"github.com/polyscone/tofu/internal/adapter/web/event"
	"github.com/polyscone/tofu/internal/adapter/web/httputil"
	"github.com/polyscone/tofu/internal/app"
	"github.com/polyscone/tofu/internal/app/account"
	"github.com/polyscone/tofu/internal/pkg/errsx"
	"github.com/polyscone/tofu/internal/pkg/http/middleware"
	"github.com/polyscone/tofu/internal/pkg/http/router"
)

func resetPasswordRoutes(h *api.Handler, mux *router.ServeMux) {
	mux.Prefix("/reset-password", func(mux *router.ServeMux) {
		mux.Post("/", resetPasswordPost(h))
		mux.Post("/new-password", resetPasswordNewPasswordPost(h))
	})
}

func resetPasswordPost(h *api.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Email string
		}
		if err := httputil.DecodeJSON(&input, r.Body); err != nil {
			h.ErrorJSON(w, r, "decode JSON", err)

			return
		}

		if _, err := account.NewEmail(input.Email); err != nil {
			err = fmt.Errorf("%w: %w", app.ErrMalformedInput, errsx.Map{
				"email": err,
			})

			h.ErrorJSON(w, r, "new email", err)

			return
		}

		h.Broker.Dispatch(event.PasswordResetRequested{
			Email: input.Email,
		})

		ctx := r.Context()

		w.Header().Set(middleware.CSRFTokenHeaderName, httputil.MaskedCSRFToken(ctx))

		w.WriteHeader(http.StatusOK)
	}
}

func resetPasswordNewPasswordPost(h *api.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Token            string
			NewPassword      string
			NewPasswordCheck string
		}
		if err := httputil.DecodeJSON(&input, r.Body); err != nil {
			h.ErrorJSON(w, r, "decode JSON", err)

			return
		}

		ctx := r.Context()

		email, err := auth.ResetPassword(ctx, h.Handler, w, r, input.Token, input.NewPassword, input.NewPasswordCheck)
		if err != nil {
			h.ErrorJSON(w, r, "reset password", err)

			return
		}

		w.Header().Set(middleware.CSRFTokenHeaderName, httputil.MaskedCSRFToken(ctx))

		h.JSON(w, r, http.StatusOK, map[string]any{
			"email": email,
		})
	}
}
