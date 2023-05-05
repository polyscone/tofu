package account

import (
	"encoding/base64"
	"net/http"

	"github.com/polyscone/tofu/internal/adapter/web/handler"
	"github.com/polyscone/tofu/internal/adapter/web/httputil"
	"github.com/polyscone/tofu/internal/adapter/web/sess"
	"github.com/polyscone/tofu/internal/adapter/web/token"
	"github.com/polyscone/tofu/internal/pkg/csrf"
	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/http/router"
	"github.com/polyscone/tofu/internal/port/account"
)

func ChangePassword(svc *handler.Services, mux *router.ServeMux, tokens token.Repo) {
	mux.Put("/password", changePasswordPut(svc))
}

func changePasswordPut(svc *handler.Services) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			OldPassword      string
			NewPassword      string
			NewPasswordCheck string
		}
		if svc.ErrorJSON(w, r, errors.Tracef(httputil.DecodeJSON(r, &input))) {
			return
		}

		ctx := r.Context()

		passport := svc.Passport(ctx)

		cmd := account.ChangePassword{
			Guard:            passport,
			UserID:           passport.GetString(sess.UserID),
			OldPassword:      input.OldPassword,
			NewPassword:      input.NewPassword,
			NewPasswordCheck: input.NewPasswordCheck,
		}
		err := cmd.Execute(ctx, svc.Bus)
		if svc.ErrorJSON(w, r, errors.Tracef(err)) {
			return
		}

		err = csrf.RenewToken(ctx)
		if svc.ErrorJSON(w, r, errors.Tracef(err)) {
			return
		}

		err = passport.Renew()
		if svc.ErrorJSON(w, r, errors.Tracef(err)) {
			return
		}

		csrfTokenBase64 := base64.RawURLEncoding.EncodeToString(csrf.MaskedToken(ctx))

		svc.JSON(w, r, map[string]any{
			"csrfToken": csrfTokenBase64,
		})
	}
}
