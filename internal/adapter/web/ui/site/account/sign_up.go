package account

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/polyscone/tofu/internal/adapter/web/handler"
	"github.com/polyscone/tofu/internal/adapter/web/httputil"
	"github.com/polyscone/tofu/internal/adapter/web/sess"
	"github.com/polyscone/tofu/internal/adapter/web/ui"
	"github.com/polyscone/tofu/internal/app"
	"github.com/polyscone/tofu/internal/pkg/background"
	"github.com/polyscone/tofu/internal/pkg/http/router"
)

func SignUp(h *ui.Handler, mux *router.ServeMux) {
	mux.Prefix("/sign-up", func(mux *router.ServeMux) {
		mux.Get("/", signUpGet(h), "account.sign_up")
		mux.Post("/", signUpPost(h), "account.sign_up.post")

		mux.Get("/success", signUpSuccessGet(h), "account.sign_up.success")
	})
}

func signUpGet(h *ui.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		if h.Sessions.GetBool(ctx, sess.IsSignedIn) {
			h.HTML.View(w, r, http.StatusOK, "site/account/sign_out/signed_in", nil)

			return
		}

		h.HTML.View(w, r, http.StatusOK, "site/account/sign_up/form", nil)
	}
}

func signUpPost(h *ui.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Email string
		}
		if err := httputil.DecodeRequestForm(&input, r); err != nil {
			h.HTML.ErrorView(w, r, "decode form", err, "site/error", nil)

			return
		}

		ctx := r.Context()
		logger := h.Logger(ctx)
		config := h.Config(ctx)

		_, err := h.Svc.Account.SignUp(ctx, input.Email)
		if err != nil {
			switch {
			case errors.Is(err, app.ErrConflictingInput):
				background.Go(func() {
					// We can't use the request context here because it will have already
					// been cancelled after the main request handler finished
					ctx := context.Background()

					tok, err := h.Repo.Web.AddResetPasswordToken(ctx, input.Email, 2*time.Hour)
					if err != nil {
						logger.Error("sign up: add reset password token: %v", "error", err)

						return
					}

					vars := handler.Vars{"Token": tok}
					if err := h.SendEmail(ctx, config.SystemEmail, input.Email, "sign_up_reset_password", vars); err != nil {
						logger.Error("sign up: send email: %v", "error", err)
					}
				})

			default:
				h.HTML.ErrorView(w, r, "sign up", err, "site/account/sign_up/form", nil)

				return
			}
		}

		h.Sessions.Set(ctx, "account.sign_up.email", input.Email)

		http.Redirect(w, r, h.Path("account.sign_up.success"), http.StatusSeeOther)
	}
}

func signUpSuccessGet(h *ui.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		h.HTML.View(w, r, http.StatusOK, "site/account/sign_up/success", handler.Vars{
			"Email": h.Sessions.PopString(ctx, "account.sign_up.email"),
		})
	}
}