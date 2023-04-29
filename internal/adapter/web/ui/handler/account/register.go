package account

import (
	"net/http"

	"github.com/polyscone/tofu/internal/adapter/web/httputil"
	"github.com/polyscone/tofu/internal/adapter/web/ui/handler"
	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/http/router"
	"github.com/polyscone/tofu/internal/pkg/valobj/uuid"
	"github.com/polyscone/tofu/internal/port/account"
)

func Register(svc *handler.Services, mux *router.ServeMux) {
	mux.Get("/register", registerGet(svc), "account/register")
	mux.Post("/register", registerPost(svc), "account/register.post")

	svc.SetViewVars("account/register", handler.Vars{
		"Email": "",
	})
}

func registerGet(svc *handler.Services) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		svc.Render(w, r, http.StatusOK, "account/register", nil)
	}
}

func registerPost(svc *handler.Services) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			UserID        string
			Email         string
			Password      string
			PasswordCheck string `form:"password"` // The UI doesn't include a check field
		}
		err := httputil.DecodeForm(r, &input)
		if svc.RenderError(w, r, errors.Tracef(err), "error", nil) {
			return
		}

		id, err := uuid.NewV4()
		if svc.RenderError(w, r, errors.Tracef(err), "error", nil) {
			return
		}
		input.UserID = id.String()

		ctx := r.Context()

		cmd := account.Register(input)
		err = cmd.Execute(ctx, svc.Bus)
		if svc.RenderError(w, r, errors.Tracef(err), "account/register", nil) {
			return
		}

		svc.Render(w, r, http.StatusOK, "account/register", handler.Vars{
			"Email": input.Email,
		})
	}
}
