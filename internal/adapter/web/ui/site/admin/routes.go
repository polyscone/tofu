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
			account.RoleManagement(h, mux)
			account.UserManagement(h, mux)
		})

		mux.Prefix("/system", func(mux *router.ServeMux) {
			system.Config(h, mux)
			system.Metrics(h, mux)
		})
	})
}
