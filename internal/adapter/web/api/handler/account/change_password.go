package account

import (
	"encoding/base64"
	"net/http"

	"github.com/polyscone/tofu/internal/adapter/web/handler"
	"github.com/polyscone/tofu/internal/adapter/web/httputil"
	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/http/router"
	"github.com/polyscone/tofu/internal/port/account"
)

func ChangePassword(svc *handler.Services, mux *router.ServeMux) {
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
			UserID:           passport.UserID(),
			OldPassword:      input.OldPassword,
			NewPassword:      input.NewPassword,
			NewPasswordCheck: input.NewPasswordCheck,
		}
		err := cmd.Execute(ctx, svc.Bus)
		if svc.ErrorJSON(w, r, errors.Tracef(err)) {
			return
		}

		csrfToken, err := svc.RenewSession(ctx)
		if svc.ErrorJSON(w, r, errors.Tracef(err)) {
			return
		}

		svc.JSON(w, r, map[string]any{
			"csrfToken": base64.RawURLEncoding.EncodeToString(csrfToken),
		})
	}
}
