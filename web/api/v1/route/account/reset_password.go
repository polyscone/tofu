package account

import (
	"fmt"
	"net/http"

	"github.com/polyscone/tofu/app"
	"github.com/polyscone/tofu/app/account"
	"github.com/polyscone/tofu/internal/errsx"
	"github.com/polyscone/tofu/internal/httpx"
	"github.com/polyscone/tofu/internal/httpx/middleware"
	"github.com/polyscone/tofu/internal/httpx/router"
	"github.com/polyscone/tofu/web/api/v1/ui"
	"github.com/polyscone/tofu/web/auth"
	"github.com/polyscone/tofu/web/event"
)

func RegisterResetPasswordHandlers(h *ui.Handler, mux *router.ServeMux) {
	mux.HandleFunc("POST /api/v1/account/reset-password", resetPasswordPost(h))
	mux.HandleFunc("POST /api/v1/account/reset-password/new-password", resetPasswordNewPasswordPost(h))
}

func resetPasswordPost(h *ui.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Email string
		}
		if err := httpx.DecodeRequestJSON(&input, r); err != nil {
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

		ctx := r.Context()

		h.Broker.Dispatch(ctx, event.PasswordResetRequested{
			Email: input.Email,
		})

		w.Header().Set(middleware.CSRFTokenHeaderName, httpx.MaskedCSRFToken(ctx))

		w.WriteHeader(http.StatusOK)
	}
}

func resetPasswordNewPasswordPost(h *ui.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Token            string
			NewPassword      string
			NewPasswordCheck string
		}
		if err := httpx.DecodeRequestJSON(&input, r); err != nil {
			h.ErrorJSON(w, r, "decode JSON", err)

			return
		}

		ctx := r.Context()

		email, err := auth.ResetPassword(ctx, h.Handler, w, r, input.Token, input.NewPassword, input.NewPasswordCheck)
		if err != nil {
			h.ErrorJSON(w, r, "reset password", err)

			return
		}

		w.Header().Set(middleware.CSRFTokenHeaderName, httpx.MaskedCSRFToken(ctx))

		h.JSON(w, r, http.StatusOK, map[string]any{
			"email": email,
		})
	}
}
