package account

import (
	"encoding/base64"
	"net/http"

	"github.com/polyscone/tofu/internal/adapter/web/event"
	"github.com/polyscone/tofu/internal/adapter/web/handler"
	"github.com/polyscone/tofu/internal/adapter/web/httputil"
	"github.com/polyscone/tofu/internal/adapter/web/token"
	"github.com/polyscone/tofu/internal/pkg/csrf"
	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/http/router"
	"github.com/polyscone/tofu/internal/pkg/valobj/text"
	"github.com/polyscone/tofu/internal/port/account"
)

func ResetPassword(svc *handler.Services, mux *router.ServeMux, tokens token.Repo) {
	mux.Post("/password/reset", resetPasswordPost(svc, tokens))
	mux.Put("/password/reset", resetPasswordPut(svc, tokens))
}

func resetPasswordPost(svc *handler.Services, tokens token.Repo) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Email string
		}
		if svc.ErrorJSON(w, r, errors.Tracef(httputil.DecodeJSON(r, &input))) {
			return
		}

		_, err := text.NewEmail(input.Email)
		if svc.ErrorJSON(w, r, errors.Tracef(err)) {
			return
		}

		svc.Broker.Dispatch(event.ResetPasswordRequested{
			Email: input.Email,
		})
	}
}

func resetPasswordPut(svc *handler.Services, tokens token.Repo) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Token            string
			NewPassword      string
			NewPasswordCheck string
		}
		if svc.ErrorJSON(w, r, errors.Tracef(httputil.DecodeJSON(r, &input))) {
			return
		}

		ctx := r.Context()

		email, err := tokens.FindResetPasswordTokenEmail(ctx, input.Token)
		if svc.ErrorJSON(w, r, errors.Tracef(err)) {
			return
		}

		passport, err := svc.PassportByEmail(ctx, email.String())
		if svc.ErrorJSON(w, r, errors.Tracef(err)) {
			return
		}

		cmd := account.ResetPassword{
			Guard:            passport,
			UserID:           passport.UserID(),
			NewPassword:      input.NewPassword,
			NewPasswordCheck: input.NewPasswordCheck,
		}
		err = cmd.Validate(ctx)
		if svc.ErrorJSON(w, r, errors.Tracef(err)) {
			return
		}

		// Only consume after manual command validation, but before execution
		// This way the token will only be consumed once we know there aren't any
		// input validation or authorisation errors
		err = tokens.ConsumeResetPasswordToken(ctx, input.Token)
		if svc.ErrorJSON(w, r, errors.Tracef(err)) {
			return
		}

		err = cmd.Execute(ctx, svc.Bus)
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
