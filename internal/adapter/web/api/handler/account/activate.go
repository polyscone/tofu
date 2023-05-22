package account

import (
	"net/http"

	"github.com/polyscone/tofu/internal/adapter/web/handler"
	"github.com/polyscone/tofu/internal/adapter/web/httputil"
	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/http/router"
	"github.com/polyscone/tofu/internal/port/account"
)

func Activate(svc *handler.Services, mux *router.ServeMux) {
	mux.Post("/activate", activatePost(svc))
}

func activatePost(svc *handler.Services) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Token string
		}
		if svc.ErrorJSON(w, r, errors.Tracef(httputil.DecodeJSON(r, &input))) {
			return
		}

		ctx := r.Context()

		email, err := svc.Web.Tokens.FindActivationTokenEmail(ctx, input.Token)
		if svc.ErrorJSON(w, r, errors.Tracef(err)) {
			return
		}

		cmd := account.Activate{
			Email: email,
		}
		err = cmd.Validate()
		if svc.ErrorJSON(w, r, errors.Tracef(err)) {
			return
		}

		// Only consume after manual command validation, but before execution
		// This way the token will only be consumed once we know there aren't any
		// input validation or authorisation errors
		err = svc.Web.Tokens.ConsumeActivationToken(ctx, input.Token)
		if svc.ErrorJSON(w, r, errors.Tracef(err)) {
			return
		}

		err = cmd.Execute(ctx, svc.Bus)
		if svc.ErrorJSON(w, r, errors.Tracef(err)) {
			return
		}
	}
}
