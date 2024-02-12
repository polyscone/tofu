package account

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/polyscone/tofu/internal/adapter/web/guard"
	"github.com/polyscone/tofu/internal/adapter/web/handler"
	"github.com/polyscone/tofu/internal/adapter/web/httputil"
	"github.com/polyscone/tofu/internal/adapter/web/sess"
	"github.com/polyscone/tofu/internal/adapter/web/ui"
	"github.com/polyscone/tofu/internal/app"
	"github.com/polyscone/tofu/internal/app/account"
	"github.com/polyscone/tofu/internal/pkg/http/router"
	"github.com/polyscone/tofu/internal/repository"
)

func RegisterRoleManagementHandlers(h *ui.Handler, mux *router.ServeMux) {
	mux.Group(func(mux *router.ServeMux) {
		mux.Before(h.CanAccess(func(p guard.Passport) bool { return p.Account.CanViewRoles() }))

		mux.HandleFunc("GET /admin/account/roles", roleListGet(h), "account.management.role.list")

		mux.Group(func(mux *router.ServeMux) {
			mux.Before(h.CanAccess(func(p guard.Passport) bool { return p.Account.CanCreateRoles() }))

			mux.HandleFunc("GET /admin/account/roles/new", roleNewGet(h), "account.management.role.new")
			mux.HandleFunc("POST /admin/account/roles/new", roleNewPost(h), "account.management.role.new.post")
		})

		mux.Group(func(mux *router.ServeMux) {
			mux.Before(h.CanAccess(func(p guard.Passport) bool { return p.Account.CanUpdateRoles() }))

			mux.HandleFunc("GET /admin/account/roles/{roleID}", roleEditGet(h), "account.management.role.edit")
			mux.HandleFunc("POST /admin/account/roles/{roleID}", roleEditPost(h), "account.management.role.edit.post")

			mux.Group(func(mux *router.ServeMux) {
				mux.Before(h.CanAccess(func(p guard.Passport) bool { return p.Account.CanDeleteRoles() }))

				mux.HandleFunc("GET /admin/account/roles/{roleID}/delete", roleDeleteGet(h), "account.management.role.delete")
				mux.HandleFunc("POST /admin/account/roles/{roleID}/delete", roleDeletePost(h), "account.management.role.delete.post")
			})
		})
	})
}

func roleListGet(h *ui.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		sortTopID := h.Sessions.PopInt(ctx, sess.SortTopID)
		sorts := r.URL.Query()["sort"]
		search := r.URL.Query().Get("search")
		page, size := httputil.Pagination(r)
		roles, total, err := h.Repo.Account.FindRolesPageBySearch(ctx, sortTopID, sorts, search, page, size)
		if err != nil {
			h.HTML.ErrorView(w, r, "find roles page by search", err, "site/error", nil)

			return
		}

		h.HTML.View(w, r, http.StatusOK, "site/account/management/role/list", handler.Vars{
			"Roles": repository.NewBook(roles, page, size, total),
			"Super": account.SuperRole,
		})
	}
}

func roleNewGet(h *ui.Handler) http.HandlerFunc {
	h.HTML.SetViewVars("site/account/management/role/new", func(r *http.Request) (handler.Vars, error) {
		vars := handler.Vars{"PermissionGroups": guard.PermissionGroups}

		return vars, nil
	})

	return func(w http.ResponseWriter, r *http.Request) {
		h.HTML.View(w, r, http.StatusOK, "site/account/management/role/new", nil)
	}
}

func roleNewPost(h *ui.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Name        string   `form:"name"`
			Description string   `form:"description"`
			Permissions []string `form:"permissions"`
		}
		if err := httputil.DecodeRequestForm(&input, r); err != nil {
			h.HTML.ErrorView(w, r, "decode form", err, "site/error", nil)

			return
		}

		ctx := r.Context()
		passport := h.Passport(ctx)

		role, err := h.Svc.Account.CreateRole(ctx, passport.Account, input.Name, input.Description, input.Permissions)
		if err != nil {
			h.HTML.ErrorView(w, r, "create role", err, "site/account/management/role/new", nil)

			return
		}

		h.AddFlashf(ctx, "Role %q created successfully.", role.Name)

		h.Sessions.Set(ctx, sess.SortTopID, role.ID)
		h.Sessions.Set(ctx, sess.HighlightID, role.ID)

		http.Redirect(w, r, h.Path("account.management.role.list"), http.StatusSeeOther)
	}
}

func roleEditGet(h *ui.Handler) http.HandlerFunc {
	h.HTML.SetViewVars("site/account/management/role/edit", func(r *http.Request) (handler.Vars, error) {
		roleID, ok := router.PathValueAs[int](r, "roleID")
		if !ok {
			return nil, errors.New("URL param as: invalid int")
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
		h.HTML.View(w, r, http.StatusOK, "site/account/management/role/edit", nil)
	}
}

func roleEditPost(h *ui.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Name        string   `form:"name"`
			Description string   `form:"description"`
			Permissions []string `form:"permissions"`
		}
		if err := httputil.DecodeRequestForm(&input, r); err != nil {
			h.HTML.ErrorView(w, r, "decode form", err, "site/error", nil)

			return
		}

		roleID, ok := router.PathValueAs[int](r, "roleID")
		if !ok {
			h.HTML.ErrorView(w, r, "URL param as", errors.New("invalid int"), "site/error", nil)

			return
		}

		if roleID == account.SuperRole.ID {
			h.HTML.ErrorView(w, r, "edit super role", app.ErrForbidden, "site/error", nil)

			return
		}

		ctx := r.Context()
		passport := h.Passport(ctx)

		role, err := h.Svc.Account.UpdateRole(ctx, passport.Account, roleID, input.Name, input.Description, input.Permissions)
		if err != nil {
			h.HTML.ErrorView(w, r, "update role", err, "site/account/management/role/edit", nil)

			return
		}

		h.AddFlashf(ctx, "Role %q updated successfully.", role.Name)

		h.Sessions.Set(ctx, sess.HighlightID, role.ID)

		http.Redirect(w, r, h.PathQuery(r, "account.management.role.list"), http.StatusSeeOther)
	}
}

func roleDeleteGet(h *ui.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		roleID, ok := router.PathValueAs[int](r, "roleID")
		if !ok {
			h.HTML.ErrorView(w, r, "URL param as", errors.New("invalid int"), "site/error", nil)

			return
		}

		if roleID == account.SuperRole.ID {
			h.HTML.ErrorView(w, r, "delete super role", app.ErrForbidden, "site/error", nil)

			return
		}

		ctx := r.Context()

		role, err := h.Repo.Account.FindRoleByID(ctx, roleID)
		if err != nil {
			h.HTML.ErrorView(w, r, "find role by id", err, "site/error", nil)

			return
		}

		userCount, err := h.Repo.Account.CountUsersByRoleID(ctx, roleID)
		if err != nil {
			h.HTML.ErrorView(w, r, "count users by role id", err, "site/error", nil)

			return
		}

		h.HTML.View(w, r, http.StatusOK, "site/account/management/role/delete", handler.Vars{
			"Role":      role,
			"UserCount": userCount,
		})
	}
}

func roleDeletePost(h *ui.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		roleID, ok := router.PathValueAs[int](r, "roleID")
		if !ok {
			h.HTML.ErrorView(w, r, "URL param as", errors.New("invalid int"), "site/error", nil)

			return
		}

		if roleID == account.SuperRole.ID {
			h.HTML.ErrorView(w, r, "delete super role", app.ErrForbidden, "site/error", nil)

			return
		}

		ctx := r.Context()
		passport := h.Passport(ctx)

		role, err := h.Svc.Account.DeleteRole(ctx, passport.Account, roleID)
		if err != nil {
			h.HTML.ErrorView(w, r, "delete role", err, "site/error", nil)

			return
		}

		h.AddFlashf(ctx, "Role %q deleted successfully.", role.Name)

		http.Redirect(w, r, h.PathQuery(r, "account.management.role.list"), http.StatusSeeOther)
	}
}
