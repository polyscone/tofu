package account

import (
	"net/http"

	"github.com/polyscone/tofu/internal/adapter/web/handler"
	"github.com/polyscone/tofu/internal/adapter/web/httputil"
	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/http/router"
	"github.com/polyscone/tofu/internal/repo"
)

func Register(svc *handler.Services, mux *router.ServeMux) {
	mux.Prefix("/register", func(mux *router.ServeMux) {
		mux.Get("/", registerGet(svc), "account.register")
		mux.Post("/", registerPost(svc), "account.register.post")

		mux.Get("/success", registerSuccessGet(svc), "account.register.success")
	})
}

func registerGet(svc *handler.Services) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		svc.View(w, r, http.StatusOK, "account/register/form", handler.Vars{
			"Email": svc.Sessions.PopString(ctx, "account.register.email"),
		})
	}
}

func registerPost(svc *handler.Services) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Email         string
			Password      string
			PasswordCheck string `form:"password"` // The UI doesn't include a check field
		}
		err := httputil.DecodeForm(&input, r)
		if svc.ErrorView(w, r, errors.Tracef(err), "error", nil) {
			return
		}

		ctx := r.Context()

		err = svc.Account.RegisterUser(ctx, input.Email, input.Password, input.PasswordCheck)
		switch {
		case errors.Is(err, repo.ErrConflict):
			svc.View(w, r, http.StatusConflict, "account/register/form", handler.Vars{
				"IsEmailInUse": true,
			})

			return

		case svc.ErrorView(w, r, errors.Tracef(err), "error", nil):
			return
		}

		svc.Sessions.Set(ctx, "account.register.email", input.Email)

		http.Redirect(w, r, svc.Path("account.register.success"), http.StatusSeeOther)
	}
}

func registerSuccessGet(svc *handler.Services) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		svc.View(w, r, http.StatusOK, "account/register/success", handler.Vars{
			"Email": svc.Sessions.PopString(ctx, "account.register.email"),
		})
	}
}
