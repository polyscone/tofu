package account

import (
	"net/http"

	"github.com/polyscone/tofu/internal/adapter/web/ui/handler"
	"github.com/polyscone/tofu/internal/pkg/csrf"
	"github.com/polyscone/tofu/internal/pkg/errors"
)

func LogoutPost(svc *handler.Services) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		err := csrf.RenewToken(ctx)
		if svc.RenderError(w, r, errors.Tracef(err), "error", nil) {
			return
		}

		svc.Sessions.Destroy(r.Context())

		http.Redirect(w, r, svc.Path("account/login"), http.StatusSeeOther)
	}

}
