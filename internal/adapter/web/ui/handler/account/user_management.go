package account

import (
	"fmt"
	"net/http"

	"github.com/polyscone/tofu/internal/adapter/web/guard"
	"github.com/polyscone/tofu/internal/adapter/web/httputil"
	"github.com/polyscone/tofu/internal/adapter/web/sess"
	"github.com/polyscone/tofu/internal/adapter/web/ui/handler"
	"github.com/polyscone/tofu/internal/app/account"
	"github.com/polyscone/tofu/internal/pkg/http/router"
	"github.com/polyscone/tofu/internal/repository"
)

func UserManagement(h *handler.Handler, mux *router.ServeMux) {
	mux.Prefix("/users", func(mux *router.ServeMux) {
		mux.Before(h.RequireAuth(func(p guard.Passport) bool { return p.CanViewUsers() }))

		mux.Get("/", userListGet(h), "account.management.user.list")

		mux.Prefix("/:userID", func(mux *router.ServeMux) {
			mux.Before(h.RequireAuth(func(p guard.Passport) bool { return p.CanEditUsers() }))

			mux.Get("/", userEditGet(h), "account.management.user.edit")
			mux.Post("/", userEditPost(h), "account.management.user.edit.post")
		})
	})
}

func userListGet(h *handler.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		sortTopID := h.Sessions.GetInt(ctx, sess.UserID)
		search := r.URL.Query().Get("search")
		page, size := httputil.Pagination(r)
		users, total, err := h.Repo.Account.FindUsersPageBySearch(ctx, sortTopID, search, page, size)
		if err != nil {
			h.ErrorView(w, r, "find users page by search", err, "error", nil)

			return
		}

		h.View(w, r, http.StatusOK, "account/management/user/list", handler.Vars{
			"Users": repository.NewBook(users, page, size, total),
		})
	}
}

func userEditGet(h *handler.Handler) http.HandlerFunc {
	h.SetViewVars("account/management/user/edit", func(r *http.Request) (handler.Vars, error) {
		userID, err := router.URLParamAs[int](r, "userID")
		if err != nil {
			return nil, fmt.Errorf("URL param as: %w", err)
		}

		ctx := r.Context()

		user, err := h.Repo.Account.FindUserByID(ctx, userID)
		if err != nil {
			return nil, fmt.Errorf("find user by id: %w", err)
		}

		var userRoleIDs []int
		if user.Roles != nil {
			userRoleIDs = make([]int, len(user.Roles))

			for i, role := range user.Roles {
				userRoleIDs[i] = role.ID
			}
		}

		roles, _, err := h.Repo.Account.FindRoles(ctx, account.SuperRole.ID)

		vars := handler.Vars{
			"User":             user,
			"UserRoleIDs":      userRoleIDs,
			"Roles":            roles,
			"SuperRole":        account.SuperRole,
			"PermissionGroups": guard.PermissionGroups,
		}

		return vars, nil
	})

	return func(w http.ResponseWriter, r *http.Request) {
		h.View(w, r, http.StatusOK, "account/management/user/edit", nil)
	}
}

func userEditPost(h *handler.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			RoleIDs []int `form:"roles"`
			Grants  []string
			Denials []string
		}
		if err := httputil.DecodeRequestForm(&input, r); err != nil {
			h.ErrorView(w, r, "decode form", err, "error", nil)

			return
		}

		userID, err := router.URLParamAs[int](r, "userID")
		if err != nil {
			h.ErrorView(w, r, "URL param as", err, "error", nil)

			return
		}

		ctx := r.Context()
		passport := h.Passport(ctx)

		user, err := h.Repo.Account.FindUserByID(ctx, userID)
		if err != nil {
			h.ErrorView(w, r, "find user by id", err, "error", nil)

			return
		}

		err = h.Account.ChangeRoles(ctx, passport, userID, input.RoleIDs, input.Grants, input.Denials)
		if err != nil {
			h.ErrorView(w, r, "change roles", err, "account/management/user/edit", nil)

			return
		}

		h.AddFlashf(ctx, "User %v updated successfully.", user.Email)

		h.Sessions.Set(ctx, sess.HighlightID, user.ID)

		http.Redirect(w, r, h.Path("account.management.user.list"), http.StatusSeeOther)
	}
}
