package account

import (
	"net/http"

	"github.com/polyscone/tofu/internal/adapter/web/httputil"
	"github.com/polyscone/tofu/internal/adapter/web/token"
	"github.com/polyscone/tofu/internal/adapter/web/ui/handler"
	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/http/router"
	"github.com/polyscone/tofu/internal/port/account"
)

func Activate(svc *handler.Services, mux *router.ServeMux, tokens token.Repo) {
	mux.Get("/activate", activateGet(svc), "account.activate")
	mux.Post("/activate", activatePost(svc, tokens), "account.activate.post")
}

func activateGet(svc *handler.Services) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		svc.Render(w, r, http.StatusOK, "account/activate", nil)
	}
}

func activatePost(svc *handler.Services, tokens token.Repo) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Token string
		}
		err := httputil.DecodeForm(r, &input)
		if svc.RenderError(w, r, errors.Tracef(err), "error", nil) {
			return
		}

		ctx := r.Context()

		if input.Token == "" {
			http.Redirect(w, r, svc.Path("account.activate"), http.StatusSeeOther)

			return
		}

		email, err := tokens.FindActivationTokenEmail(ctx, input.Token)
		if svc.RenderError(w, r, errors.Tracef(err), "account/activate", nil) {
			return
		}

		cmd := account.Activate{
			Email: email.String(),
		}
		err = cmd.Validate(ctx)
		if svc.RenderError(w, r, errors.Tracef(err), "account/activate", nil) {
			return
		}

		// Only consume after manual command validation, but before execution
		// This way the token will only be consumed once we know there aren't any
		// input validation or authorisation errors
		err = tokens.ConsumeActivationToken(ctx, input.Token)
		if svc.RenderError(w, r, errors.Tracef(err), "account/activate", nil) {
			return
		}

		err = cmd.Execute(ctx, svc.Bus)
		if svc.RenderError(w, r, errors.Tracef(err), "account/activate", nil) {
			return
		}

		http.Redirect(w, r, svc.Path("account.activate")+"?status=success", http.StatusSeeOther)
	}
}
