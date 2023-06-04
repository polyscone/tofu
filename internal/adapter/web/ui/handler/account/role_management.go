package account

import (
	"net/http"

	"github.com/polyscone/tofu/internal/adapter/web/handler"
	"github.com/polyscone/tofu/internal/adapter/web/httputil"
	"github.com/polyscone/tofu/internal/adapter/web/passport"
	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/http/router"
	"github.com/polyscone/tofu/internal/repo"
)

func RoleManagement(svc *handler.Services, mux *router.ServeMux, guard *handler.Guard) {
	mux.Prefix("/roles", func(mux *router.ServeMux) {
		mux.Get("/", roleListGet(svc), "account.management.role.list")

		mux.Prefix("/new", func(mux *router.ServeMux) {
			guard.RequireAuthPrefix(mux.CurrentPath(), func(p passport.Passport) bool {
				return p.CanCreateRoles()
			})

			mux.Get("/", roleNewGet(svc), "account.management.role.new")
			mux.Post("/", roleNewPost(svc), "account.management.role.new.post")
		})

		mux.Prefix("/:roleID", func(mux *router.ServeMux) {
			mux.Get("/", roleEditGet(svc), "account.management.role.edit")
			mux.Post("/", roleEditPost(svc), "account.management.role.edit.post")

			mux.Prefix("/delete", func(mux *router.ServeMux) {
				mux.Get("/", roleDeleteGet(svc), "account.management.role.delete")
				mux.Post("/", roleDeletePost(svc), "account.management.role.delete.post")
			})
		})
	})
}

func roleListGet(svc *handler.Services) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		sortTopID := svc.Sessions.PopInt(ctx, "role.sort_top_id")
		highlightID := svc.Sessions.PopInt(ctx, "role.highlight_id")
		search := r.URL.Query().Get("search")
		page, size := svc.Pagination(r)
		roles, total, err := svc.Repo.Account.FindRolesPageBySearch(ctx, sortTopID, search, page, size)
		if svc.ErrorView(w, r, errors.Tracef(err), "error", nil) {
			return
		}

		svc.View(w, r, http.StatusOK, "account/management/role/list", handler.Vars{
			"HighlightID": highlightID,
			"Roles":       repo.NewBook(roles, page, size, total),
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
		var input struct {
			Name string
		}
		err := httputil.DecodeForm(&input, r)
		if svc.ErrorView(w, r, errors.Tracef(err), "error", nil) {
			return
		}

		ctx := r.Context()

		passport := svc.Passport(ctx)

		role, err := svc.Account.CreateRole(ctx, passport, input.Name)
		if svc.ErrorView(w, r, errors.Tracef(err), "account/management/role/new", nil) {
			return
		}

		svc.AddFlashf(ctx, "Role %q created successfully.", role.Name)

		svc.Sessions.Set(ctx, "role.sort_top_id", role.ID)
		svc.Sessions.Set(ctx, "role.highlight_id", role.ID)

		http.Redirect(w, r, svc.Path("account.management.role.list"), http.StatusSeeOther)
	}
}

func roleEditGet(svc *handler.Services) http.HandlerFunc {
	svc.SetViewVars("account/management/role/edit", func(r *http.Request) (handler.Vars, error) {
		roleID, err := router.URLParamAs[int](r, "roleID")
		if err != nil {
			return nil, errors.Tracef(err)
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
		var input struct {
			Name string
		}
		err := httputil.DecodeForm(&input, r)
		if svc.ErrorView(w, r, errors.Tracef(err), "error", nil) {
			return
		}

		roleID, err := router.URLParamAs[int](r, "roleID")
		if svc.ErrorView(w, r, errors.Tracef(err), "error", nil) {
			return
		}

		ctx := r.Context()

		passport := svc.Passport(ctx)

		role, err := svc.Account.EditRole(ctx, passport, roleID, input.Name)
		if svc.ErrorView(w, r, errors.Tracef(err), "account/management/role/edit", nil) {
			return
		}

		svc.AddFlashf(ctx, "Role %q updated successfully.", role.Name)

		http.Redirect(w, r, svc.PathQuery(r, "account.management.role.list"), http.StatusSeeOther)
	}
}

func roleDeleteGet(svc *handler.Services) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		roleID, err := router.URLParamAs[int](r, "roleID")
		if svc.ErrorView(w, r, errors.Tracef(err), "error", nil) {
			return
		}

		ctx := r.Context()

		role, err := svc.Repo.Account.FindRoleByID(ctx, roleID)
		if svc.ErrorView(w, r, errors.Tracef(err), "error", nil) {
			return
		}

		svc.View(w, r, http.StatusOK, "account/management/role/delete", handler.Vars{
			"Role": role,
		})
	}
}

func roleDeletePost(svc *handler.Services) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		roleID, err := router.URLParamAs[int](r, "roleID")
		if svc.ErrorView(w, r, errors.Tracef(err), "error", nil) {
			return
		}

		ctx := r.Context()

		passport := svc.Passport(ctx)

		role, err := svc.Account.DeleteRole(ctx, passport, roleID)
		if svc.ErrorView(w, r, errors.Tracef(err), "error", nil) {
			return
		}

		svc.AddFlashf(ctx, "Role %q deleted successfully.", role.Name)

		http.Redirect(w, r, svc.PathQuery(r, "account.management.role.list"), http.StatusSeeOther)
	}
}
