package account

import (
	"fmt"
	"net/http"

	"github.com/polyscone/tofu/internal/adapter/web/guard"
	"github.com/polyscone/tofu/internal/adapter/web/handler"
	"github.com/polyscone/tofu/internal/adapter/web/httputil"
	"github.com/polyscone/tofu/internal/adapter/web/sess"
	"github.com/polyscone/tofu/internal/app"
	"github.com/polyscone/tofu/internal/app/account"
	"github.com/polyscone/tofu/internal/pkg/http/router"
	"github.com/polyscone/tofu/internal/repository"
)

func RoleManagement(h *handler.Handler, mux *router.ServeMux) {
	mux.Prefix("/roles", func(mux *router.ServeMux) {
		mux.Before(h.RequireAuth(func(p guard.Passport) bool { return p.Account.CanViewRoles() }))

		mux.Get("/", roleListGet(h), "account.management.role.list")

		mux.Prefix("/new", func(mux *router.ServeMux) {
			mux.Before(h.RequireAuth(func(p guard.Passport) bool { return p.Account.CanCreateRoles() }))

			mux.Get("/", roleNewGet(h), "account.management.role.new")
			mux.Post("/", roleNewPost(h), "account.management.role.new.post")
		})

		mux.Prefix("/:roleID", func(mux *router.ServeMux) {
			mux.Before(h.RequireAuth(func(p guard.Passport) bool { return p.Account.CanUpdateRoles() }))

			mux.Get("/", roleEditGet(h), "account.management.role.edit")
			mux.Post("/", roleEditPost(h), "account.management.role.edit.post")

			mux.Prefix("/delete", func(mux *router.ServeMux) {
				mux.Before(h.RequireAuth(func(p guard.Passport) bool { return p.Account.CanDeleteRoles() }))

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
		search := r.URL.Query().Get("search")
		page, size := httputil.Pagination(r)
		roles, total, err := h.Repo.Account.FindRolesPageBySearch(ctx, sortTopID, search, page, size)
		if err != nil {
			h.HTML.ErrorView(w, r, "find roles page by search", err, "error", nil)

			return
		}

		h.HTML.View(w, r, http.StatusOK, "account/management/role/list", handler.Vars{
			"Roles": repository.NewBook(roles, page, size, total),
			"Super": account.SuperRole,
		})
	}
}

func roleNewGet(h *handler.Handler) http.HandlerFunc {
	h.SetViewVars("account/management/role/new", func(r *http.Request) (handler.Vars, error) {
		vars := handler.Vars{
			"PermissionGroups": guard.PermissionGroups,
		}

		return vars, nil
	})

	return func(w http.ResponseWriter, r *http.Request) {
		h.HTML.View(w, r, http.StatusOK, "account/management/role/new", nil)
	}
}

func roleNewPost(h *handler.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Name        string
			Description string
			Permissions []string
		}
		if err := httputil.DecodeRequestForm(&input, r); err != nil {
			h.HTML.ErrorView(w, r, "decode form", err, "error", nil)

			return
		}

		ctx := r.Context()
		passport := h.Passport(ctx)

		role, err := h.Account.CreateRole(ctx, passport.Account, input.Name, input.Description, input.Permissions)
		if err != nil {
			h.HTML.ErrorView(w, r, "create role", err, "account/management/role/new", nil)

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
			return nil, fmt.Errorf("URL param as: %w", err)
		}

		if roleID == account.SuperRole.ID {
			return nil, fmt.Errorf("edit super role: %w", app.ErrForbidden)
		}

		ctx := r.Context()

		role, err := h.Repo.Account.FindRoleByID(ctx, roleID)
		if err != nil {
			return nil, fmt.Errorf("find role by id: %w", err)
		}

		vars := handler.Vars{
			"Role":             role,
			"PermissionGroups": guard.PermissionGroups,
		}

		return vars, nil
	})

	return func(w http.ResponseWriter, r *http.Request) {
		h.HTML.View(w, r, http.StatusOK, "account/management/role/edit", nil)
	}
}

func roleEditPost(h *handler.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Name        string
			Description string
			Permissions []string
		}
		if err := httputil.DecodeRequestForm(&input, r); err != nil {
			h.HTML.ErrorView(w, r, "decode form", err, "error", nil)

			return
		}

		roleID, err := router.URLParamAs[int](r, "roleID")
		if err != nil {
			h.HTML.ErrorView(w, r, "URL param as", err, "error", nil)

			return
		}

		if roleID == account.SuperRole.ID {
			h.HTML.ErrorView(w, r, "edit super role", app.ErrForbidden, "error", nil)

			return
		}

		ctx := r.Context()
		passport := h.Passport(ctx)

		role, err := h.Account.UpdateRole(ctx, passport.Account, roleID, input.Name, input.Description, input.Permissions)
		if err != nil {
			h.HTML.ErrorView(w, r, "update role", err, "account/management/role/edit", nil)

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
			h.HTML.ErrorView(w, r, "URL param as", err, "error", nil)

			return
		}

		if roleID == account.SuperRole.ID {
			h.HTML.ErrorView(w, r, "delete super role", app.ErrForbidden, "error", nil)

			return
		}

		ctx := r.Context()

		role, err := h.Repo.Account.FindRoleByID(ctx, roleID)
		if err != nil {
			h.HTML.ErrorView(w, r, "find role by id", err, "error", nil)

			return
		}

		userCount, err := h.Repo.Account.CountUsersByRoleID(ctx, roleID)
		if err != nil {
			h.HTML.ErrorView(w, r, "count users by role id", err, "error", nil)

			return
		}

		h.HTML.View(w, r, http.StatusOK, "account/management/role/delete", handler.Vars{
			"Role":      role,
			"UserCount": userCount,
		})
	}
}

func roleDeletePost(h *handler.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		roleID, err := router.URLParamAs[int](r, "roleID")
		if err != nil {
			h.HTML.ErrorView(w, r, "URL param as", err, "error", nil)

			return
		}

		if roleID == account.SuperRole.ID {
			h.HTML.ErrorView(w, r, "delete super role", app.ErrForbidden, "error", nil)

			return
		}

		ctx := r.Context()
		passport := h.Passport(ctx)

		role, err := h.Account.DeleteRole(ctx, passport.Account, roleID)
		if err != nil {
			h.HTML.ErrorView(w, r, "delete role", err, "error", nil)

			return
		}

		h.AddFlashf(ctx, "Role %q deleted successfully.", role.Name)

		http.Redirect(w, r, h.PathQuery(r, "account.management.role.list"), http.StatusSeeOther)
	}
}
