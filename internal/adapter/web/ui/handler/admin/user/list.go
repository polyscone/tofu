package user

import (
	"net/http"

	"github.com/polyscone/tofu/internal/adapter/web/handler"
	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/http/router"
)

func List(svc *handler.Services, mux *router.ServeMux) {
	mux.Get("/list", listGet(svc), "admin.user.list")

	svc.SetViewVars("admin/user/list", handler.Vars{
		"User":  nil,
		"Users": nil,
	})
}

func listGet(svc *handler.Services) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		if editID := r.URL.Query().Get("edit"); editID != "" {
			user, err := svc.Account.Users.FindByID(ctx, editID)
			if svc.ErrorView(w, r, errors.Tracef(err), "error", nil) {
				return
			}

			svc.View(w, r, http.StatusOK, "admin/user/list", handler.Vars{
				"User": user,
			})
		} else {
			search := r.URL.Query().Get("search")
			page, size := svc.Pagination(r)
			users, err := svc.Account.Users.FindByPage(ctx, page, size, search)
			if svc.ErrorView(w, r, errors.Tracef(err), "error", nil) {
				return
			}

			svc.View(w, r, http.StatusOK, "admin/user/list", handler.Vars{
				"Users": users,
			})
		}
	}
}
