package account

import (
	"net/http"

	"github.com/polyscone/tofu/internal/adapter/web/handler"
	"github.com/polyscone/tofu/internal/adapter/web/httputil"
	"github.com/polyscone/tofu/internal/adapter/web/sess"
	"github.com/polyscone/tofu/internal/adapter/web/ui"
	"github.com/polyscone/tofu/internal/app"
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
		config := h.Config(ctx)

		if !config.SignUpEnabled {
			h.HTML.ErrorView(w, r, "sign up", app.ErrForbidden, "site/error", nil)

			return
		}

		_, err := h.Svc.Account.SignUp(ctx, input.Email)
		if err != nil {
			h.HTML.ErrorView(w, r, "sign up", err, "site/account/sign_up/form", nil)

			return
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
