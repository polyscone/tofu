package account

import (
	"context"
	"net/http"
	"time"

	"github.com/polyscone/tofu/internal/adapter/web/handler"
	"github.com/polyscone/tofu/internal/adapter/web/httputil"
	"github.com/polyscone/tofu/internal/adapter/web/sess"
	"github.com/polyscone/tofu/internal/app"
	"github.com/polyscone/tofu/internal/pkg/background"
	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/http/router"
	"github.com/polyscone/tofu/internal/pkg/logger"
)

func SignUp(svc *handler.Services, mux *router.ServeMux) {
	mux.Prefix("/sign-up", func(mux *router.ServeMux) {
		mux.Get("/", signUpGet(svc), "account.sign_up")
		mux.Post("/", signUpPost(svc), "account.sign_up.post")

		mux.Get("/success", signUpSuccessGet(svc), "account.sign_up.success")
	})
}

func signUpGet(svc *handler.Services) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		if svc.Sessions.GetBool(ctx, sess.IsSignedIn) {
			svc.View(w, r, http.StatusOK, "account/sign_out/signed_in", nil)

			return
		}

		svc.View(w, r, http.StatusOK, "account/sign_up/form", nil)
	}
}

func signUpPost(svc *handler.Services) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Email string
		}
		err := httputil.DecodeForm(&input, r)
		if svc.ErrorView(w, r, errors.Tracef(err), "error", nil) {
			return
		}

		ctx := r.Context()

		_, err = svc.Account.SignUp(ctx, input.Email)
		if err != nil {
			switch {
			case errors.Is(err, app.ErrConflictingInput):
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
					if err := svc.SendEmail(ctx, recipients, "sign_up_reset_password", vars); err != nil {
						logger.PrintError(err)
					}
				})

			default:
				svc.ErrorView(w, r, errors.Tracef(err), "account/sign_up/form", nil)

				return
			}
		}

		svc.Sessions.Set(ctx, "account.sign_up.email", input.Email)

		http.Redirect(w, r, svc.Path("account.sign_up.success"), http.StatusSeeOther)
	}
}

func signUpSuccessGet(svc *handler.Services) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		svc.View(w, r, http.StatusOK, "account/sign_up/success", handler.Vars{
			"Email": svc.Sessions.PopString(ctx, "account.sign_up.email"),
		})
	}
}
