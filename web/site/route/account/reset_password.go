package account

import (
	"fmt"
	"net/http"

	"github.com/polyscone/tofu/app"
	"github.com/polyscone/tofu/app/account"
	"github.com/polyscone/tofu/internal/errsx"
	"github.com/polyscone/tofu/internal/httpx"
	"github.com/polyscone/tofu/internal/httpx/router"
	"github.com/polyscone/tofu/internal/i18n"
	"github.com/polyscone/tofu/web/auth"
	"github.com/polyscone/tofu/web/event"
	"github.com/polyscone/tofu/web/site/ui"
)

func RegisterResetPasswordHandlers(h *ui.Handler, mux *router.ServeMux) {
	mux.HandleFunc("GET /account/reset-password", resetPasswordGet(h), "account.reset_password")
	mux.HandleFunc("POST /account/reset-password", resetPasswordPost(h), "account.reset_password.post")

	mux.HandleFunc("GET /account/reset-password/email-sent", resetPasswordEmailSentGet(h), "account.reset_password.email_sent")

	mux.HandleFunc("GET /account/reset-password/new-password", resetPasswordNewPasswordGet(h), "account.reset_password.new_password")
	mux.HandleFunc("POST /account/reset-password/new-password", resetPasswordNewPasswordPost(h), "account.reset_password.new_password.post")
}

func resetPasswordGet(h *ui.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h.HTML.View(w, r, http.StatusOK, "account/reset_password/request", nil)
	}
}

func resetPasswordPost(h *ui.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Email string `form:"email"`
		}
		if err := httpx.DecodeRequestForm(&input, r); err != nil {
			h.HTML.ErrorView(w, r, "decode form", err, "error", nil)

			return
		}

		ctx := r.Context()

		if _, err := account.NewEmail(input.Email); err != nil {
			err = fmt.Errorf("%w: %w", app.ErrMalformedInput, errsx.Map{
				"email": err,
			})

			h.HTML.ErrorView(w, r, "new email", err, h.Session.LastView(ctx), nil)

			return
		}

		h.Broker.Dispatch(ctx, event.PasswordResetRequested{
			Email: input.Email,
		})

		http.Redirect(w, r, h.Path("account.reset_password.email_sent"), http.StatusSeeOther)
	}
}

func resetPasswordEmailSentGet(h *ui.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h.HTML.View(w, r, http.StatusOK, "account/reset_password/email_sent", nil)
	}
}

func resetPasswordNewPasswordGet(h *ui.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h.HTML.View(w, r, http.StatusOK, "account/reset_password/new_password", nil)
	}
}

func resetPasswordNewPasswordPost(h *ui.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Token            string `form:"token"`
			NewPassword      string `form:"new-password"`
			NewPasswordCheck string `form:"new-password"` // The UI doesn't include a check field
		}
		if err := httpx.DecodeRequestForm(&input, r); err != nil {
			h.HTML.ErrorView(w, r, "decode form", err, "error", nil)

			return
		}

		ctx := r.Context()

		email, err := auth.ResetPassword(ctx, h.Handler, w, r, input.Token, input.NewPassword, input.NewPasswordCheck)
		if err != nil {
			h.HTML.ErrorView(w, r, "reset password", err, h.Session.LastView(ctx), nil)

			return
		}

		h.AddFlashf(ctx, i18n.M("site.account.reset_password.flash.password_changed"))

		signInWithPassword(ctx, h, w, r, email, input.NewPassword)
	}
}
