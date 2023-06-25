package account

import (
	"net/http"

	"github.com/polyscone/tofu/internal/adapter/web/handler"
	"github.com/polyscone/tofu/internal/adapter/web/httputil"
	"github.com/polyscone/tofu/internal/adapter/web/sess"
	"github.com/polyscone/tofu/internal/pkg/http/router"
	"github.com/polyscone/tofu/internal/pkg/password/pwned"
)

func ChoosePassword(h *handler.Handler, mux *router.ServeMux) {
	mux.Prefix("/choose-password", func(mux *router.ServeMux) {
		mux.Before(h.RequireSignIn)
		mux.Before(func(w http.ResponseWriter, r *http.Request) bool {
			ctx := r.Context()
			user := h.User(ctx)

			if len(user.HashedPassword) != 0 {
				http.Redirect(w, r, h.Path("account.change_password"), http.StatusSeeOther)

				return false
			}

			return true
		})

		mux.Get("/", h.HTML.Handler("site/account/choose_password/form"), "account.choose_password")
		mux.Post("/", choosePasswordPost(h), "account.choose_password.post")
	})
}

func choosePasswordPost(h *handler.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			NewPassword      string
			NewPasswordCheck string `form:"new-password"` // The UI doesn't include a check field
		}
		if err := httputil.DecodeRequestForm(&input, r); err != nil {
			h.HTML.ErrorView(w, r, "decode form", err, "site/error", nil)

			return
		}

		ctx := r.Context()
		log := h.Logger(ctx)
		user := h.User(ctx)
		passport := h.Passport(ctx)

		err := h.Account.ChoosePassword(ctx,
			passport.Account,
			user.ID,
			input.NewPassword,
			input.NewPasswordCheck,
		)
		if err != nil {
			h.HTML.ErrorView(w, r, "choose password", err, "site/account/choose_password/form", nil)

			return
		}

		if _, err := h.RenewSession(ctx); err != nil {
			h.HTML.ErrorView(w, r, "renew session", err, "site/error", nil)

			return
		}

		knownBreachCount, err := pwned.KnownPasswordBreachCount(ctx, []byte(input.NewPassword))
		if err != nil {
			log.Error("known password breach count", "error", err)

			h.Sessions.Delete(ctx, sess.KnownPasswordBreachCount)
		} else {
			if knownBreachCount > 0 {
				h.Sessions.Set(ctx, sess.KnownPasswordBreachCount, knownBreachCount)
			} else {
				h.Sessions.Delete(ctx, sess.KnownPasswordBreachCount)
			}
		}

		h.AddFlashf(ctx, "Your password has been successfully set.")

		var redirect string
		if r := h.Sessions.PopString(ctx, sess.Redirect); r != "" {
			redirect = r
		} else {
			redirect = h.Path("account.change_password")
		}

		http.Redirect(w, r, redirect, http.StatusSeeOther)
	}
}
