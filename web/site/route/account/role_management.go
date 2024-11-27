package account

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/polyscone/tofu/app"
	"github.com/polyscone/tofu/internal/collection"
	"github.com/polyscone/tofu/internal/httpx"
	"github.com/polyscone/tofu/internal/httpx/router"
	"github.com/polyscone/tofu/internal/i18n"
	"github.com/polyscone/tofu/web/guard"
	"github.com/polyscone/tofu/web/handler"
	"github.com/polyscone/tofu/web/site/ui"
)

func RegisterRoleManagementHandlers(h *ui.Handler, mux *router.ServeMux) {
	mux.Group(func(mux *router.ServeMux) {
		mux.Before(h.RequireSignIn)

		mux.Group(func(mux *router.ServeMux) {
			mux.Before(h.CanAccess(func(p guard.Passport) bool { return p.Account.CanViewRoles() }))

			mux.HandleFunc("GET /admin/account/roles", roleListGet(h), "account.management.role.list")
		})

		mux.Group(func(mux *router.ServeMux) {
			mux.Before(h.CanAccess(func(p guard.Passport) bool { return p.Account.CanCreateRoles() }))

			mux.HandleFunc("GET /admin/account/roles/new", roleNewGet(h), "account.management.role.new")
			mux.HandleFunc("POST /admin/account/roles/new", roleNewPost(h), "account.management.role.new.post")
		})

		mux.Group(func(mux *router.ServeMux) {
			mux.Before(h.CanAccess(func(p guard.Passport) bool { return p.Account.CanUpdateRoles() }))

			mux.HandleFunc("GET /admin/account/roles/{roleID}", roleEditGet(h), "account.management.role.edit")
			mux.HandleFunc("POST /admin/account/roles/{roleID}", roleEditPost(h), "account.management.role.edit.post")
		})

		mux.Group(func(mux *router.ServeMux) {
			mux.Before(h.CanAccess(func(p guard.Passport) bool { return p.Account.CanDeleteRoles() }))

			mux.HandleFunc("GET /admin/account/roles/{roleID}/delete", roleDeleteGet(h), "account.management.role.delete")
			mux.HandleFunc("POST /admin/account/roles/{roleID}/delete", roleDeletePost(h), "account.management.role.delete.post")
		})
	})
}

func roleListGet(h *ui.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		sortTopID := h.Session.PopSortTopID(ctx)
		sorts := r.URL.Query()["sort"]
		search := r.URL.Query().Get("search")
		page, size := httpx.Pagination(r)
		roles, total, err := h.Repo.Account.FindRolesPageBySearch(ctx, page, size, sortTopID, sorts, search)
		if err != nil {
			h.HTML.ErrorView(w, r, "find roles page by search", err, "error", nil)

			return
		}

		h.HTML.View(w, r, http.StatusOK, "account/management/role/list", handler.Vars{
			"Roles": collection.NewBook(roles, page, size, total),
			"Super": h.SuperRole,
		})
	}
}

func roleNewGet(h *ui.Handler) http.HandlerFunc {
	const view = "account/management/role/new"
	h.HTML.SetViewVars(view, func(r *http.Request) (handler.Vars, error) {
		vars := handler.Vars{"PermissionGroups": guard.PermissionGroups}

		return vars, nil
	})

	return func(w http.ResponseWriter, r *http.Request) {
		h.HTML.View(w, r, http.StatusOK, view, nil)
	}
}

func roleNewPost(h *ui.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Name        string   `form:"name"`
			Description string   `form:"description"`
			Permissions []string `form:"permissions"`
		}
		if err := httpx.DecodeRequestForm(&input, r); err != nil {
			h.HTML.ErrorView(w, r, "decode form", err, "error", nil)

			return
		}

		ctx := r.Context()
		passport := h.Passport(ctx)

		role, err := h.Svc.Account.CreateRole(ctx, passport.Account, input.Name, input.Description, input.Permissions)
		if err != nil {
			h.HTML.ErrorView(w, r, "create role", err, h.Session.LastView(ctx), nil)

			return
		}

		h.AddFlashf(ctx, i18n.M("site.account.role_management.flash.created", "name", role.Name))

		h.Session.SetSortTopID(ctx, role.ID)
		h.Session.SetHighlightID(ctx, role.ID)

		http.Redirect(w, r, h.Path("account.management.role.list"), http.StatusSeeOther)
	}
}

func roleEditGet(h *ui.Handler) http.HandlerFunc {
	const view = "account/management/role/edit"
	h.HTML.SetViewVars(view, func(r *http.Request) (handler.Vars, error) {
		roleID, _ := strconv.Atoi(r.PathValue("roleID"))
		if roleID == h.SuperRole.ID {
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
		h.HTML.View(w, r, http.StatusOK, view, nil)
	}
}

func roleEditPost(h *ui.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Name        string   `form:"name"`
			Description string   `form:"description"`
			Permissions []string `form:"permissions"`
		}
		if err := httpx.DecodeRequestForm(&input, r); err != nil {
			h.HTML.ErrorView(w, r, "decode form", err, "error", nil)

			return
		}

		roleID, _ := strconv.Atoi(r.PathValue("roleID"))
		if roleID == h.SuperRole.ID {
			h.HTML.ErrorView(w, r, "edit super role", app.ErrForbidden, "error", nil)

			return
		}

		ctx := r.Context()
		passport := h.Passport(ctx)

		role, err := h.Svc.Account.UpdateRole(ctx, passport.Account, roleID, input.Name, input.Description, input.Permissions)
		if err != nil {
			h.HTML.ErrorView(w, r, "update role", err, h.Session.LastView(ctx), nil)

			return
		}

		h.AddFlashf(ctx, i18n.M("site.account.role_management.flash.updated", "name", role.Name))

		h.Session.SetHighlightID(ctx, role.ID)

		http.Redirect(w, r, h.PathQuery(r, "account.management.role.list"), http.StatusSeeOther)
	}
}

func roleDeleteGet(h *ui.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		roleID, _ := strconv.Atoi(r.PathValue("roleID"))
		if roleID == h.SuperRole.ID {
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

func roleDeletePost(h *ui.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		roleID, _ := strconv.Atoi(r.PathValue("roleID"))
		if roleID == h.SuperRole.ID {
			h.HTML.ErrorView(w, r, "delete super role", app.ErrForbidden, "error", nil)

			return
		}

		ctx := r.Context()
		passport := h.Passport(ctx)

		role, err := h.Svc.Account.DeleteRole(ctx, passport.Account, roleID)
		if err != nil {
			h.HTML.ErrorView(w, r, "delete role", err, "error", nil)

			return
		}

		h.AddFlashf(ctx, i18n.M("site.account.role_management.flash.deleted", "name", role.Name))

		http.Redirect(w, r, h.PathQuery(r, "account.management.role.list"), http.StatusSeeOther)
	}
}
