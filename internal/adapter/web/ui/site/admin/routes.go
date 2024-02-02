package admin

import (
	"github.com/polyscone/tofu/internal/adapter/web/ui"
	"github.com/polyscone/tofu/internal/adapter/web/ui/site/account"
	"github.com/polyscone/tofu/internal/adapter/web/ui/site/system"
	"github.com/polyscone/tofu/internal/pkg/http/router"
)

func Routes(h *ui.Handler, mux *router.ServeMux) {
	mux.Prefix("/admin", func(mux *router.ServeMux) {
		mux.Name("admin.section")

		mux.Before(h.RequireSignIn)

		mux.Get("/", h.HTML.Handler("site/admin/dashboard"), "admin.dashboard")

		mux.Prefix("/account", func(mux *router.ServeMux) {
			account.RoleManagementRoutes(h, mux)
			account.UserManagementRoutes(h, mux)
		})

		mux.Prefix("/system", func(mux *router.ServeMux) {
			system.ConfigRoutes(h, mux)
			system.MetricsRoutes(h, mux)
		})
	})
}
