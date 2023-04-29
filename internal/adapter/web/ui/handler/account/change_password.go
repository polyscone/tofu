package account

import (
	"net/http"

	"github.com/polyscone/tofu/internal/adapter/web/httputil"
	"github.com/polyscone/tofu/internal/adapter/web/sess"
	"github.com/polyscone/tofu/internal/adapter/web/ui/handler"
	"github.com/polyscone/tofu/internal/pkg/csrf"
	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/http/router"
	"github.com/polyscone/tofu/internal/port/account"
)

func ChangePassword(svc *handler.Services, mux *router.ServeMux) {
	mux.Get("/change-password", changePasswordGet(svc), "account/change_password")
	mux.Put("/change-password", changePasswordPut(svc), "account/change_password.put")
}

func changePasswordGet(svc *handler.Services) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		svc.Render(w, r, http.StatusOK, "account/change_password", nil)
	}
}

func changePasswordPut(svc *handler.Services) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			OldPassword      string
			NewPassword      string
			NewPasswordCheck string `form:"new-password"` // The UI doesn't include a check field
		}
		err := httputil.DecodeForm(r, &input)
		if svc.RenderError(w, r, errors.Tracef(err), "error", nil) {
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
		err = cmd.Execute(ctx, svc.Bus)
		if svc.RenderError(w, r, errors.Tracef(err), "account/change_password", nil) {
			return
		}

		err = csrf.RenewToken(ctx)
		if svc.RenderError(w, r, errors.Tracef(err), "error", nil) {
			return
		}

		err = passport.Renew()
		if svc.RenderError(w, r, errors.Tracef(err), "error", nil) {
			return
		}

		http.Redirect(w, r, svc.Path("account/change_password")+"?status=success", http.StatusSeeOther)
	}
}
