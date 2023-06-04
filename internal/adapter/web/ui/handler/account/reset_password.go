package account

import (
	"context"
	"net/http"
	"time"

	"github.com/polyscone/tofu/internal/adapter/web/handler"
	"github.com/polyscone/tofu/internal/adapter/web/httputil"
	"github.com/polyscone/tofu/internal/pkg/background"
	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/http/router"
	"github.com/polyscone/tofu/internal/pkg/logger"
	"github.com/polyscone/tofu/internal/pkg/valobj/text"
)

func ResetPassword(svc *handler.Services, mux *router.ServeMux) {
	mux.Prefix("/reset-password", func(mux *router.ServeMux) {
		mux.Get("/", resetPasswordGet(svc), "account.reset_password")
		mux.Post("/", resetPasswordPost(svc), "account.reset_password.post")

		mux.Get("/email-sent", resetPasswordEmailSentGet(svc), "account.reset_password.email_sent")

		mux.Get("/new-password", resetPasswordNewPasswordGet(svc), "account.reset_password.new_password")
		mux.Post("/new-password", resetPasswordNewPasswordPost(svc), "account.reset_password.new_password.post")
	})
}

func resetPasswordGet(svc *handler.Services) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		svc.View(w, r, http.StatusOK, "account/reset_password/request", nil)
	}
}

func resetPasswordPost(svc *handler.Services) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Email string
		}
		if svc.ErrorView(w, r, errors.Tracef(httputil.DecodeForm(&input, r)), "error", nil) {
			return
		}

		if _, err := text.NewEmail(input.Email); err != nil {
			svc.ErrorViewFunc(w, r, errors.Tracef(err), "account/reset_password/request", func(data *handler.ViewData) {
				data.Errors = errors.Map{"email": err}
			})

			return
		}

		background.Go(func() {
			ctx := context.Background()

			tok, err := svc.Repo.Web.AddResetPasswordToken(ctx, input.Email, 2*time.Hour)
			if err != nil {
				logger.PrintError(err)

				return
			}

			recipients := handler.EmailRecipients{
				From: svc.Email.From,
				To:   []string{input.Email},
			}
			vars := handler.Vars{
				"Token": tok,
			}
			if err := svc.SendEmail(ctx, recipients, "reset_password", vars); err != nil {
				logger.PrintError(err)
			}
		})

		http.Redirect(w, r, svc.Path("account.reset_password.email_sent"), http.StatusSeeOther)
	}
}

func resetPasswordEmailSentGet(svc *handler.Services) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		svc.View(w, r, http.StatusOK, "account/reset_password/email_sent", nil)
	}
}

func resetPasswordNewPasswordGet(svc *handler.Services) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		svc.View(w, r, http.StatusOK, "account/reset_password/new_password", nil)
	}
}

func resetPasswordNewPasswordPost(svc *handler.Services) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Token            string
			NewPassword      string
			NewPasswordCheck string `form:"new-password"` // The UI doesn't include a check field
		}
		err := httputil.DecodeForm(&input, r)
		if svc.ErrorView(w, r, errors.Tracef(err), "error", nil) {
			return
		}

		ctx := r.Context()

		email, err := svc.Repo.Web.FindResetPasswordTokenEmail(ctx, input.Token)
		if svc.ErrorView(w, r, errors.Tracef(err), "error", nil) {
			return
		}

		passport, err := svc.PassportByEmail(ctx, email)
		if svc.ErrorView(w, r, errors.Tracef(err), "error", nil) {
			return
		}

		err = svc.Account.ResetPassword(ctx, passport, passport.UserID(), input.NewPassword, input.NewPasswordCheck)
		if svc.ErrorView(w, r, errors.Tracef(err), "account/reset_password/new_password", nil) {
			return
		}

		err = svc.Repo.Web.ConsumeResetPasswordToken(ctx, input.Token)
		if svc.ErrorView(w, r, errors.Tracef(err), "error", nil) {
			return
		}

		svc.AddFlashf(ctx, "Your password has been successfully changed.")

		signInWithPassword(ctx, svc, w, r, email, input.NewPassword)
	}
}
