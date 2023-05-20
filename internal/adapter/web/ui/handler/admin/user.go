package admin

import (
	"github.com/polyscone/tofu/internal/adapter/web/handler"
	"github.com/polyscone/tofu/internal/adapter/web/ui/handler/admin/user"
	"github.com/polyscone/tofu/internal/pkg/http/router"
)

func UserManagement(svc *handler.Services, mux *router.ServeMux) {
	mux.Prefix("/users", func(mux *router.ServeMux) {
		user.List(svc, mux)
	})
}
