package account

import (
	"context"
	"net/http"
	"time"

	"github.com/polyscone/tofu/internal/adapter/web/httputil"
	"github.com/polyscone/tofu/internal/adapter/web/sess"
	"github.com/polyscone/tofu/internal/adapter/web/ui/handler"
	"github.com/polyscone/tofu/internal/app/account"
	"github.com/polyscone/tofu/internal/pkg/background"
	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/http/router"
	"github.com/polyscone/tofu/internal/pkg/logger"
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
		if h.ErrorView(w, r, errors.Tracef(httputil.DecodeForm(&input, r)), "error", nil) {
			return
		}

		if _, err := account.NewEmail(input.Email); err != nil {
			h.ErrorViewFunc(w, r, errors.Tracef(err), "account/reset_password/request", func(data *handler.ViewData) {
				data.Errors = errors.Map{"email": err}
			})

			return
		}

		background.Go(func() {
			ctx := context.Background()

			tok, err := h.Store.Web.AddResetPasswordToken(ctx, input.Email, 2*time.Hour)
			if err != nil {
				logger.PrintError(errors.Tracef(err))

				return
			}

			recipients := handler.EmailRecipients{
				From: h.Email.From,
				To:   []string{input.Email},
			}
			vars := handler.Vars{
				"Token": tok,
			}
			if err := h.SendEmail(ctx, recipients, "reset_password", vars); err != nil {
				logger.PrintError(errors.Tracef(err))
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
		err := httputil.DecodeForm(&input, r)
		if h.ErrorView(w, r, errors.Tracef(err), "error", nil) {
			return
		}

		ctx := r.Context()

		email, err := h.Store.Web.FindResetPasswordTokenEmail(ctx, input.Token)
		if h.ErrorView(w, r, errors.Tracef(err), "error", nil) {
			return
		}

		passport, err := h.PassportByEmail(ctx, email)
		if h.ErrorView(w, r, errors.Tracef(err), "error", nil) {
			return
		}

		userID := h.Sessions.GetInt(ctx, sess.UserID)

		err = h.Account.ResetPassword(ctx, passport, userID, input.NewPassword, input.NewPasswordCheck)
		if h.ErrorView(w, r, errors.Tracef(err), "account/reset_password/new_password", nil) {
			return
		}

		err = h.Store.Web.ConsumeResetPasswordToken(ctx, input.Token)
		if h.ErrorView(w, r, errors.Tracef(err), "error", nil) {
			return
		}

		h.AddFlashf(ctx, "Your password has been successfully changed.")

		signInWithPassword(ctx, h, w, r, email, input.NewPassword)
	}
}
