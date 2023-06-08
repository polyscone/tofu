package account

import (
	"context"
	"net/http"
	"time"

	"github.com/polyscone/tofu/internal/adapter/web/httputil"
	"github.com/polyscone/tofu/internal/adapter/web/sess"
	"github.com/polyscone/tofu/internal/adapter/web/ui/handler"
	"github.com/polyscone/tofu/internal/app"
	"github.com/polyscone/tofu/internal/pkg/background"
	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/http/router"
	"github.com/polyscone/tofu/internal/pkg/logger"
)

func SignUp(h *handler.Handler, mux *router.ServeMux) {
	mux.Prefix("/sign-up", func(mux *router.ServeMux) {
		mux.Get("/", signUpGet(h), "account.sign_up")
		mux.Post("/", signUpPost(h), "account.sign_up.post")

		mux.Get("/success", signUpSuccessGet(h), "account.sign_up.success")
	})
}

func signUpGet(h *handler.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		if h.Sessions.GetBool(ctx, sess.IsSignedIn) {
			h.View(w, r, http.StatusOK, "account/sign_out/signed_in", nil)

			return
		}

		h.View(w, r, http.StatusOK, "account/sign_up/form", nil)
	}
}

func signUpPost(h *handler.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Email string
		}
		err := httputil.DecodeForm(&input, r)
		if h.ErrorView(w, r, errors.Tracef(err), "error", nil) {
			return
		}

		ctx := r.Context()

		_, err = h.Account.SignUp(ctx, input.Email)
		if err != nil {
			switch {
			case errors.Is(err, app.ErrConflictingInput):
				background.Go(func() {
					ctx := context.Background()

					tok, err := h.Store.Web.AddResetPasswordToken(ctx, input.Email, 2*time.Hour)
					if err != nil {
						logger.PrintError(err)

						return
					}

					recipients := handler.EmailRecipients{
						From: h.Email.From,
						To:   []string{input.Email},
					}
					vars := handler.Vars{
						"Token": tok,
					}
					if err := h.SendEmail(ctx, recipients, "sign_up_reset_password", vars); err != nil {
						logger.PrintError(err)
					}
				})

			default:
				h.ErrorView(w, r, errors.Tracef(err), "account/sign_up/form", nil)

				return
			}
		}

		h.Sessions.Set(ctx, "account.sign_up.email", input.Email)

		http.Redirect(w, r, h.Path("account.sign_up.success"), http.StatusSeeOther)
	}
}

func signUpSuccessGet(h *handler.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		h.View(w, r, http.StatusOK, "account/sign_up/success", handler.Vars{
			"Email": h.Sessions.PopString(ctx, "account.sign_up.email"),
		})
	}
}
