package admin

import (
	"github.com/polyscone/tofu/internal/adapter/web/guard"
	"github.com/polyscone/tofu/internal/adapter/web/ui"
	"github.com/polyscone/tofu/internal/adapter/web/ui/site/account"
	"github.com/polyscone/tofu/internal/pkg/http/router"
)

func Routes(h *ui.Handler, mux *router.ServeMux) {
	mux.Prefix("/admin", func(mux *router.ServeMux) {
		mux.Name("admin.section")

		dashboardRoutes(h, mux)

		mux.Prefix("/account", func(mux *router.ServeMux) {
			mux.Before(h.RequireSignIn)

			account.RoleManagement(h, mux)
			account.UserManagement(h, mux)
		})

		mux.Prefix("/system", func(mux *router.ServeMux) {
			mux.Before(h.RequireSignInIf(func(p guard.Passport) bool { return !p.System.CanViewConfig() }))
			mux.Before(h.CanAccess(func(p guard.Passport) bool { return p.System.CanViewConfig() }))

			systemConfigRoutes(h, mux)
		})
	})
}
