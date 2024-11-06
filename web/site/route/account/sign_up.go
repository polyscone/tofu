package account

import (
	"errors"
	"net/http"

	"github.com/polyscone/tofu/app"
	"github.com/polyscone/tofu/internal/httpx"
	"github.com/polyscone/tofu/internal/httpx/router"
	"github.com/polyscone/tofu/web/auth"
	"github.com/polyscone/tofu/web/handler"
	"github.com/polyscone/tofu/web/site/ui"
)

func RegisterSignUpHandlers(h *ui.Handler, mux *router.ServeMux) {
	mux.HandleFunc("GET /account/sign-up", signUpGet(h), "account.sign_up")
	mux.HandleFunc("POST /account/sign-up", signUpPost(h), "account.sign_up.post")

	mux.HandleFunc("GET /account/sign-up/success", signUpSuccessGet(h), "account.sign_up.success")
}

func signUpGet(h *ui.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		config := h.Config(ctx)

		if !config.SignUpEnabled {
			h.HTML.ErrorView(w, r, "sign up", app.ErrNotFound, "error", nil)

			return
		}

		if h.Session.IsSignedIn(ctx) {
			h.HTML.View(w, r, http.StatusOK, "account/sign_out/signed_in", nil)

			return
		}

		h.HTML.View(w, r, http.StatusOK, "account/sign_up/form", nil)
	}
}

func signUpPost(h *ui.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Email string `form:"email"`
		}
		if err := httpx.DecodeRequestForm(&input, r); err != nil {
			h.HTML.ErrorView(w, r, "decode form", err, "error", nil)

			return
		}

		ctx := r.Context()

		if err := auth.SignUp(ctx, h.Handler, w, r, input.Email); err != nil {
			if errors.Is(err, app.ErrForbidden) {
				h.HTML.ErrorView(w, r, "sign up", err, "error", nil)
			} else {
				h.HTML.ErrorView(w, r, "sign up", err, h.Session.LastView(ctx), nil)
			}

			return
		}

		h.Session.Set(ctx, "account.sign_up.email", input.Email)

		http.Redirect(w, r, h.Path("account.sign_up.success"), http.StatusSeeOther)
	}
}

func signUpSuccessGet(h *ui.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		h.HTML.View(w, r, http.StatusOK, "account/sign_up/success", handler.Vars{
			"Email": h.Session.PopString(ctx, "account.sign_up.email"),
		})
	}
}
