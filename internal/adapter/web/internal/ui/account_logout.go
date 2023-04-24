package ui

import (
	"net/http"

	"github.com/polyscone/tofu/internal/pkg/csrf"
	"github.com/polyscone/tofu/internal/pkg/errors"
)

func (ui *UI) accountLogoutPost(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	err := csrf.RenewToken(ctx)
	if ui.renderError(w, r, errors.Tracef(err)) {
		return
	}

	ui.sessions.Destroy(r.Context())

	http.Redirect(w, r, "/account/login", http.StatusSeeOther)
}
