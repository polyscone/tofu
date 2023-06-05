package account

import (
	"net/http"

	"github.com/polyscone/tofu/internal/adapter/web/httputil"
	"github.com/polyscone/tofu/internal/adapter/web/passport"
	"github.com/polyscone/tofu/internal/adapter/web/ui/handler"
	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/http/router"
	"github.com/polyscone/tofu/internal/repo"
)

func RoleManagement(h *handler.Handler, guard *handler.Guard, mux *router.ServeMux) {
	mux.Prefix("/roles", func(mux *router.ServeMux) {
		mux.Get("/", roleListGet(h), "account.management.role.list")

		mux.Prefix("/new", func(mux *router.ServeMux) {
			guard.RequireAuth(mux.CurrentPrefix(), func(p passport.Passport) bool {
				return p.CanCreateRoles()
			})

			mux.Get("/", roleNewGet(h), "account.management.role.new")
			mux.Post("/", roleNewPost(h), "account.management.role.new.post")
		})

		mux.Prefix("/:roleID", func(mux *router.ServeMux) {
			mux.Get("/", roleEditGet(h), "account.management.role.edit")
			mux.Post("/", roleEditPost(h), "account.management.role.edit.post")

			mux.Prefix("/delete", func(mux *router.ServeMux) {
				mux.Get("/", roleDeleteGet(h), "account.management.role.delete")
				mux.Post("/", roleDeletePost(h), "account.management.role.delete.post")
			})
		})
	})
}

func roleListGet(h *handler.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		sortTopID := h.Sessions.PopInt(ctx, "role.sort_top_id")
		highlightID := h.Sessions.PopInt(ctx, "role.highlight_id")
		search := r.URL.Query().Get("search")
		page, size := httputil.Pagination(r)
		roles, total, err := h.Repo.Account.FindRolesPageBySearch(ctx, sortTopID, search, page, size)
		if h.ErrorView(w, r, errors.Tracef(err), "error", nil) {
			return
		}

		h.View(w, r, http.StatusOK, "account/management/role/list", handler.Vars{
			"HighlightID": highlightID,
			"Roles":       repo.NewBook(roles, page, size, total),
		})
	}
}

func roleNewGet(h *handler.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h.View(w, r, http.StatusOK, "account/management/role/new", nil)
	}
}

func roleNewPost(h *handler.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Name string
		}
		err := httputil.DecodeForm(&input, r)
		if h.ErrorView(w, r, errors.Tracef(err), "error", nil) {
			return
		}

		ctx := r.Context()

		passport := h.Passport(ctx)

		role, err := h.Account.CreateRole(ctx, passport, input.Name)
		if h.ErrorView(w, r, errors.Tracef(err), "account/management/role/new", nil) {
			return
		}

		h.AddFlashf(ctx, "Role %q created successfully.", role.Name)

		h.Sessions.Set(ctx, "role.sort_top_id", role.ID)
		h.Sessions.Set(ctx, "role.highlight_id", role.ID)

		http.Redirect(w, r, h.Path("account.management.role.list"), http.StatusSeeOther)
	}
}

func roleEditGet(h *handler.Handler) http.HandlerFunc {
	h.SetViewVars("account/management/role/edit", func(r *http.Request) (handler.Vars, error) {
		roleID, err := router.URLParamAs[int](r, "roleID")
		if err != nil {
			return nil, errors.Tracef(httputil.ErrNotFound, err)
		}

		ctx := r.Context()

		role, err := h.Repo.Account.FindRoleByID(ctx, roleID)
		if err != nil {
			return nil, errors.Tracef(err)
		}

		vars := handler.Vars{
			"Role": role,
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
			Name string
		}
		err := httputil.DecodeForm(&input, r)
		if h.ErrorView(w, r, errors.Tracef(err), "error", nil) {
			return
		}

		roleID, err := router.URLParamAs[int](r, "roleID")
		if h.ErrorView(w, r, errors.Tracef(err), "error", nil) {
			return
		}

		ctx := r.Context()

		passport := h.Passport(ctx)

		role, err := h.Account.EditRole(ctx, passport, roleID, input.Name)
		if h.ErrorView(w, r, errors.Tracef(err), "account/management/role/edit", nil) {
			return
		}

		h.AddFlashf(ctx, "Role %q updated successfully.", role.Name)

		h.Sessions.Set(ctx, "role.highlight_id", role.ID)

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

		ctx := r.Context()

		role, err := h.Repo.Account.FindRoleByID(ctx, roleID)
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
