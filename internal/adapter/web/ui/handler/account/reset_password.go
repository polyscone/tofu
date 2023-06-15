package account

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/polyscone/tofu/internal/adapter/web/httputil"
	"github.com/polyscone/tofu/internal/adapter/web/ui/handler"
	"github.com/polyscone/tofu/internal/app"
	"github.com/polyscone/tofu/internal/app/account"
	"github.com/polyscone/tofu/internal/pkg/background"
	"github.com/polyscone/tofu/internal/pkg/errsx"
	"github.com/polyscone/tofu/internal/pkg/http/router"
)

func ResetPassword(h *handler.Handler, mux *router.ServeMux) {
	mux.Prefix("/reset-password", func(mux *router.ServeMux) {
		mux.Get("/", resetPasswordGet(h), "account.reset_password")
		mux.Post("/", resetPasswordPost(h), "account.reset_password.post")

		mux.Get("/email-sent", resetPasswordEmailSentGet(h), "account.reset_password.email_sent")

		mux.Get("/new-password", resetPasswordNewPasswordGet(h), "account.reset_password.new_password")
		mux.Post("/new-password", resetPasswordNewPasswordPost(h), "account.reset_password.new_password.post")
	})
}

func resetPasswordGet(h *handler.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h.View(w, r, http.StatusOK, "account/reset_password/request", nil)
	}
}

func resetPasswordPost(h *handler.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Email string
		}
		if err := httputil.DecodeForm(&input, r); err != nil {
			h.ErrorView(w, r, "decode form", err, "error", nil)

			return
		}

		if _, err := account.NewEmail(input.Email); err != nil {
			err = fmt.Errorf("%w: %w", app.ErrMalformedInput, errsx.Map{
				"email": err,
			})

			h.ErrorView(w, r, "new email", err, "account/reset_password/request", nil)

			return
		}

		ctx := r.Context()
		config := h.Config(ctx)

		background.Go(func() {
			// We can't use the request context here because it will have already
			// been cancelled after the main request handler finished
			ctx := context.Background()

			tok, err := h.Repo.Web.AddResetPasswordToken(ctx, input.Email, 2*time.Hour)
			if err != nil {
				h.Logger.Error("reset password: add reset password token", "error", err)

				return
			}

			recipients := handler.EmailRecipients{
				From: config.SystemEmail,
				To:   []string{input.Email},
			}
			vars := handler.Vars{
				"Token": tok,
			}
			if err := h.SendEmail(ctx, recipients, "reset_password", vars); err != nil {
				h.Logger.Error("reset password: send email", "error", err)
			}
		})

		http.Redirect(w, r, h.Path("account.reset_password.email_sent"), http.StatusSeeOther)
	}
}

func resetPasswordEmailSentGet(h *handler.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h.View(w, r, http.StatusOK, "account/reset_password/email_sent", nil)
	}
}

func resetPasswordNewPasswordGet(h *handler.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h.View(w, r, http.StatusOK, "account/reset_password/new_password", nil)
	}
}

func resetPasswordNewPasswordPost(h *handler.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Token            string
			NewPassword      string
			NewPasswordCheck string `form:"new-password"` // The UI doesn't include a check field
		}
		if err := httputil.DecodeForm(&input, r); err != nil {
			h.ErrorView(w, r, "decode form", err, "error", nil)

			return
		}

		ctx := r.Context()

		email, err := h.Repo.Web.FindResetPasswordTokenEmail(ctx, input.Token)
		if err != nil {
			h.ErrorView(w, r, "find reset password token email", err, "error", nil)

			return
		}

		user, err := h.Repo.Account.FindUserByEmail(ctx, email)
		if err != nil {
			h.ErrorView(w, r, "find user by email", err, "error", nil)

			return
		}

		passport, err := h.PassportByEmail(ctx, email)
		if err != nil {
			h.ErrorView(w, r, "passport by email", err, "error", nil)

			return
		}

		err = h.Account.ResetPassword(ctx, passport, user.ID, input.NewPassword, input.NewPasswordCheck)
		if err != nil {
			h.ErrorView(w, r, "reset password", err, "account/reset_password/new_password", nil)

			return
		}

		err = h.Repo.Web.ConsumeResetPasswordToken(ctx, input.Token)
		if err != nil {
			h.ErrorView(w, r, "consume reset password token", err, "error", nil)

			return
		}

		h.AddFlashf(ctx, "Your password has been successfully changed.")

		signInWithPassword(ctx, h, w, r, email, input.NewPassword)
	}
}
