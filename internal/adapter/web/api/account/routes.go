package account

import (
	"github.com/polyscone/tofu/internal/adapter/web/api"
	"github.com/polyscone/tofu/internal/pkg/http/router"
)

func Routes(h *api.Handler, mux *router.ServeMux) {
	mux.Prefix("/account", func(mux *router.ServeMux) {
		resetPasswordRoutes(h, mux)
		sessionRoutes(h, mux)
		signInRoutes(h, mux)
		signOutRoutes(h, mux)
		signUpRoutes(h, mux)
		verifyRoutes(h, mux)
	})
}
