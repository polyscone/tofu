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

		mux.Prefix("/new", func(mux *router.ServeMux) {
			mux.Get("/", roleNewGet(svc), "account.management.role.new")
			mux.Post("/", roleNewPost(svc), "account.management.role.new.post")
		})

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

func roleNewGet(svc *handler.Services) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		svc.View(w, r, http.StatusOK, "account/management/role/new", nil)
	}
}

func roleNewPost(svc *handler.Services) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		svc.View(w, r, http.StatusOK, "account/management/role/new", nil)
	}
}

func roleEditGet(svc *handler.Services) http.HandlerFunc {
	svc.SetViewVars("account/management/role/edit", func(r *http.Request) (handler.Vars, error) {
		roleID, err := router.URLParamAs[int](r, "roleID")
		if err != nil {
			return nil, errors.Tracef(err, httputil.ErrNotFound)
		}

		ctx := r.Context()

		role, err := svc.Repo.Account.FindRoleByID(ctx, roleID)
		if err != nil {
			return nil, errors.Tracef(err)
		}

		vars := handler.Vars{
			"Role": role,
		}

		return vars, nil
	})

	return func(w http.ResponseWriter, r *http.Request) {
		svc.View(w, r, http.StatusOK, "account/management/role/edit", nil)
	}
}

func roleEditPost(svc *handler.Services) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		svc.View(w, r, http.StatusOK, "account/management/role/edit", nil)
	}
}
