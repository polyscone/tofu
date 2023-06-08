package account

import (
	"net/http"

	"github.com/polyscone/tofu/internal/adapter/web/httputil"
	"github.com/polyscone/tofu/internal/adapter/web/passport"
	"github.com/polyscone/tofu/internal/adapter/web/sess"
	"github.com/polyscone/tofu/internal/adapter/web/ui/handler"
	"github.com/polyscone/tofu/internal/app"
	"github.com/polyscone/tofu/internal/app/account"
	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/http/router"
	"github.com/polyscone/tofu/internal/repo"
)

func RoleManagement(h *handler.Handler, mux *router.ServeMux) {
	mux.Prefix("/roles", func(mux *router.ServeMux) {
		mux.Before(h.RequireAuth(func(p passport.Passport) bool { return p.CanViewRoles() }))

		mux.Get("/", roleListGet(h), "account.management.role.list")

		mux.Prefix("/new", func(mux *router.ServeMux) {
			mux.Before(h.RequireAuth(func(p passport.Passport) bool { return p.CanCreateRoles() }))

			mux.Get("/", roleNewGet(h), "account.management.role.new")
			mux.Post("/", roleNewPost(h), "account.management.role.new.post")
		})

		mux.Prefix("/:roleID", func(mux *router.ServeMux) {
			mux.Before(h.RequireAuth(func(p passport.Passport) bool { return p.CanEditRoles() }))

			mux.Get("/", roleEditGet(h), "account.management.role.edit")
			mux.Post("/", roleEditPost(h), "account.management.role.edit.post")

			mux.Prefix("/delete", func(mux *router.ServeMux) {
				mux.Before(h.RequireAuth(func(p passport.Passport) bool { return p.CanDeleteRoles() }))

				mux.Get("/", roleDeleteGet(h), "account.management.role.delete")
				mux.Post("/", roleDeletePost(h), "account.management.role.delete.post")
			})
		})
	})
}

func roleListGet(h *handler.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		sortTopID := h.Sessions.PopInt(ctx, sess.SortTopID)
		if sortTopID == 0 {
			sortTopID = account.SuperRole.ID
		}

		search := r.URL.Query().Get("search")
		page, size := httputil.Pagination(r)
		roles, total, err := h.Store.Account.FindRolesPageBySearch(ctx, sortTopID, search, page, size)
		if h.ErrorView(w, r, errors.Tracef(err), "error", nil) {
			return
		}

		h.View(w, r, http.StatusOK, "account/management/role/list", handler.Vars{
			"Roles": repo.NewBook(roles, page, size, total),
			"Super": account.SuperRole,
		})
	}
}

func roleNewGet(h *handler.Handler) http.HandlerFunc {
	h.SetViewVars("account/management/role/new", func(r *http.Request) (handler.Vars, error) {
		vars := handler.Vars{
			"PermissionGroups": passport.PermissionGroups,
		}

		return vars, nil
	})

	return func(w http.ResponseWriter, r *http.Request) {
		h.View(w, r, http.StatusOK, "account/management/role/new", nil)
	}
}

func roleNewPost(h *handler.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Name        string
			Description string
			Permissions []string
		}
		err := httputil.DecodeForm(&input, r)
		if h.ErrorView(w, r, errors.Tracef(err), "error", nil) {
			return
		}

		ctx := r.Context()

		passport := h.Passport(ctx)

		role, err := h.Account.CreateRole(ctx, passport, input.Name, input.Description, input.Permissions)
		if h.ErrorView(w, r, errors.Tracef(err), "account/management/role/new", nil) {
			return
		}

		h.AddFlashf(ctx, "Role %q created successfully.", role.Name)

		h.Sessions.Set(ctx, sess.SortTopID, role.ID)
		h.Sessions.Set(ctx, sess.HighlightID, role.ID)

		http.Redirect(w, r, h.Path("account.management.role.list"), http.StatusSeeOther)
	}
}

func roleEditGet(h *handler.Handler) http.HandlerFunc {
	h.SetViewVars("account/management/role/edit", func(r *http.Request) (handler.Vars, error) {
		roleID, err := router.URLParamAs[int](r, "roleID")
		if err != nil {
			return nil, errors.Tracef(httputil.ErrNotFound, err)
		}

		if roleID == account.SuperRole.ID {
			return nil, errors.Tracef(app.ErrForbidden)
		}

		ctx := r.Context()

		role, err := h.Store.Account.FindRoleByID(ctx, roleID)
		if err != nil {
			return nil, errors.Tracef(err)
		}

		vars := handler.Vars{
			"Role":             role,
			"PermissionGroups": passport.PermissionGroups,
		}

		return vars, nil
	})

	return func(w http.ResponseWriter, r *http.Request) {
		h.View(w, r, http.StatusOK, "account/management/role/edit", nil)
	}
}

func roleEditPost(h *handler.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Name        string
			Description string
			Permissions []string
		}
		err := httputil.DecodeForm(&input, r)
		if h.ErrorView(w, r, errors.Tracef(err), "error", nil) {
			return
		}

		roleID, err := router.URLParamAs[int](r, "roleID")
		if h.ErrorView(w, r, errors.Tracef(err), "error", nil) {
			return
		}

		if roleID == account.SuperRole.ID {
			h.ErrorView(w, r, errors.Tracef(app.ErrForbidden), "error", nil)

			return
		}

		ctx := r.Context()

		passport := h.Passport(ctx)

		role, err := h.Account.EditRole(ctx, passport, roleID, input.Name, input.Description, input.Permissions)
		if h.ErrorView(w, r, errors.Tracef(err), "account/management/role/edit", nil) {
			return
		}

		h.AddFlashf(ctx, "Role %q updated successfully.", role.Name)

		h.Sessions.Set(ctx, sess.HighlightID, role.ID)

		http.Redirect(w, r, h.PathQuery(r, "account.management.role.list"), http.StatusSeeOther)
	}
}

func roleDeleteGet(h *handler.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		roleID, err := router.URLParamAs[int](r, "roleID")
		if err != nil {
			err = errors.Tracef(httputil.ErrNotFound, err)
		}
		if h.ErrorView(w, r, errors.Tracef(err), "error", nil) {
			return
		}

		if roleID == account.SuperRole.ID {
			h.ErrorView(w, r, errors.Tracef(app.ErrForbidden), "error", nil)

			return
		}

		ctx := r.Context()

		role, err := h.Store.Account.FindRoleByID(ctx, roleID)
		if h.ErrorView(w, r, errors.Tracef(err), "error", nil) {
			return
		}

		h.View(w, r, http.StatusOK, "account/management/role/delete", handler.Vars{
			"Role": role,
		})
	}
}

func roleDeletePost(h *handler.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		roleID, err := router.URLParamAs[int](r, "roleID")
		if h.ErrorView(w, r, errors.Tracef(err), "error", nil) {
			return
		}

		if roleID == account.SuperRole.ID {
			h.ErrorView(w, r, errors.Tracef(app.ErrForbidden), "error", nil)

			return
		}

		ctx := r.Context()

		passport := h.Passport(ctx)

		role, err := h.Account.DeleteRole(ctx, passport, roleID)
		if h.ErrorView(w, r, errors.Tracef(err), "error", nil) {
			return
		}

		h.AddFlashf(ctx, "Role %q deleted successfully.", role.Name)

		http.Redirect(w, r, h.PathQuery(r, "account.management.role.list"), http.StatusSeeOther)
	}
}
