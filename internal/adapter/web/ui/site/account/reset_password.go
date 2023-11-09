package account

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/polyscone/tofu/internal/adapter/web/handler"
	"github.com/polyscone/tofu/internal/adapter/web/httputil"
	"github.com/polyscone/tofu/internal/adapter/web/ui"
	"github.com/polyscone/tofu/internal/app"
	"github.com/polyscone/tofu/internal/app/account"
	"github.com/polyscone/tofu/internal/pkg/background"
	"github.com/polyscone/tofu/internal/pkg/errsx"
	"github.com/polyscone/tofu/internal/pkg/http/router"
)

func resetPasswordRoutes(h *ui.Handler, mux *router.ServeMux) {
	mux.Prefix("/reset-password", func(mux *router.ServeMux) {
		mux.Get("/", h.HTML.Handler("site/account/reset_password/request"), "account.reset_password")
		mux.Post("/", resetPasswordPost(h), "account.reset_password.post")

		mux.Get("/email-sent", h.HTML.Handler("site/account/reset_password/email_sent"), "account.reset_password.email_sent")

		mux.Get("/new-password", h.HTML.Handler("site/account/reset_password/new_password"), "account.reset_password.new_password")
		mux.Post("/new-password", resetPasswordNewPasswordPost(h), "account.reset_password.new_password.post")
	})
}

func resetPasswordPost(h *ui.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Email string `form:"email"`
		}
		if err := httputil.DecodeRequestForm(&input, r); err != nil {
			h.HTML.ErrorView(w, r, "decode form", err, "site/error", nil)

			return
		}

		if _, err := account.NewEmail(input.Email); err != nil {
			err = fmt.Errorf("%w: %w", app.ErrMalformedInput, errsx.Map{
				"email": err,
			})

			h.HTML.ErrorView(w, r, "new email", err, "site/account/reset_password/request", nil)

			return
		}

		ctx := r.Context()
		logger := h.Logger(ctx)
		config := h.Config(ctx)

		background.Go(func() {
			// We can't use the request context here because it will have already
			// been cancelled after the main request handler finished
			ctx := context.Background()

			tok, err := h.Repo.Web.AddResetPasswordToken(ctx, input.Email, 2*time.Hour)
			if err != nil {
				logger.Error("reset password: add reset password token", "error", err)

				return
			}

			vars := handler.Vars{"Token": tok}
			if err := h.SendEmail(ctx, config.SystemEmail, input.Email, "reset_password", vars); err != nil {
				logger.Error("reset password: send email", "error", err)
			}
		})

		http.Redirect(w, r, h.Path("account.reset_password.email_sent"), http.StatusSeeOther)
	}
}

func resetPasswordNewPasswordPost(h *ui.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Token            string `form:"token"`
			NewPassword      string `form:"new-password"`
			NewPasswordCheck string `form:"new-password"` // The UI doesn't include a check field
		}
		if err := httputil.DecodeRequestForm(&input, r); err != nil {
			h.HTML.ErrorView(w, r, "decode form", err, "site/error", nil)

			return
		}

		ctx := r.Context()

		email, err := h.Repo.Web.FindResetPasswordTokenEmail(ctx, input.Token)
		if err != nil {
			h.HTML.ErrorView(w, r, "find reset password token email", err, "site/error", nil)

			return
		}

		user, err := h.Repo.Account.FindUserByEmail(ctx, email)
		if err != nil {
			h.HTML.ErrorView(w, r, "find user by email", err, "site/error", nil)

			return
		}

		passport, err := h.PassportByEmail(ctx, email)
		if err != nil {
			h.HTML.ErrorView(w, r, "passport by email", err, "site/error", nil)

			return
		}

		err = h.Svc.Account.ResetPassword(ctx, passport.Account, user.ID, input.NewPassword, input.NewPasswordCheck)
		if err != nil {
			h.HTML.ErrorView(w, r, "reset password", err, "site/account/reset_password/new_password", nil)

			return
		}

		err = h.Repo.Web.ConsumeResetPasswordToken(ctx, input.Token)
		if err != nil {
			h.HTML.ErrorView(w, r, "consume reset password token", err, "site/error", nil)

			return
		}

		h.AddFlashf(ctx, "Your password has been successfully changed.")

		signInWithPassword(ctx, h, w, r, email, input.NewPassword)
	}
}
