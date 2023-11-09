package account

import (
	"github.com/polyscone/tofu/internal/adapter/web/ui"
	"github.com/polyscone/tofu/internal/pkg/http/router"
)

func Routes(h *ui.Handler, mux *router.ServeMux) {
	mux.Prefix("/account", func(mux *router.ServeMux) {
		mux.Name("account.section")

		verifyRoutes(h, mux)
		changePasswordRoutes(h, mux)
		choosePasswordRoutes(h, mux)
		dashboardRoutes(h, mux)
		resetPasswordRoutes(h, mux)
		signUpRoutes(h, mux)
		signInRoutes(h, mux)
		signOutRoutes(h, mux)
		totpRoutes(h, mux)
	})
}
