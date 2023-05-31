package account

import (
	"net/http"

	"github.com/polyscone/tofu/internal/adapter/web/handler"
	"github.com/polyscone/tofu/internal/adapter/web/httputil"
	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/http/router"
	"github.com/polyscone/tofu/internal/repo"
)

func RoleManagement(svc *handler.Services, mux *router.ServeMux) {
	mux.Prefix("/roles", func(mux *router.ServeMux) {
		mux.Get("/", roleListGet(svc), "account.management.role.list")

		mux.Prefix("/:roleID", func(mux *router.ServeMux) {
			mux.Get("/", roleEditGet(svc), "account.management.role.edit")
			mux.Post("/", roleEditPost(svc), "account.management.role.edit.post")
		})
	})
}

func roleListGet(svc *handler.Services) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		search := r.URL.Query().Get("search")
		page, size := svc.Pagination(r)
		roles, total, err := svc.Repo.Account.FindRolesPageBySearch(ctx, search, page, size)
		if svc.ErrorView(w, r, errors.Tracef(err), "error", nil) {
			return
		}

		svc.View(w, r, http.StatusOK, "account/management/role/list", handler.Vars{
			"Roles": repo.NewBook(roles, page, size, total),
		})
	}
}

func roleEditGet(svc *handler.Services) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		roleID, err := router.URLParamAs[int](r, "roleID")
		if svc.ErrorView(w, r, errors.Tracef(err, httputil.ErrNotFound), "error", nil) {
			return
		}

		ctx := r.Context()

		role, err := svc.Repo.Account.FindRoleByID(ctx, roleID)
		if svc.ErrorView(w, r, errors.Tracef(err), "error", nil) {
			return
		}

		svc.View(w, r, http.StatusOK, "account/management/role/edit", handler.Vars{
			"Role": role,
		})
	}
}

func roleEditPost(svc *handler.Services) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		roleID, err := router.URLParamAs[int](r, "roleID")
		if svc.ErrorView(w, r, errors.Tracef(err, httputil.ErrNotFound), "error", nil) {
			return
		}

		ctx := r.Context()

		role, err := svc.Repo.Account.FindRoleByID(ctx, roleID)
		if svc.ErrorView(w, r, errors.Tracef(err), "error", nil) {
			return
		}

		svc.View(w, r, http.StatusOK, "account/management/role/edit", handler.Vars{
			"Role": role,
		})
	}
}
