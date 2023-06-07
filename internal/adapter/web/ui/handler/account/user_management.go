package account

import (
	"net/http"

	"github.com/polyscone/tofu/internal/adapter/web/httputil"
	"github.com/polyscone/tofu/internal/adapter/web/sess"
	"github.com/polyscone/tofu/internal/adapter/web/ui/handler"
	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/http/router"
	"github.com/polyscone/tofu/internal/repo"
)

func UserManagement(h *handler.Handler, guard *handler.Guard, mux *router.ServeMux) {
	mux.Prefix("/users", func(mux *router.ServeMux) {
		mux.Get("/", userListGet(h), "account.management.user.list")

		mux.Prefix("/:userID", func(mux *router.ServeMux) {
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
		if h.ErrorView(w, r, errors.Tracef(err), "error", nil) {
			return
		}

		h.View(w, r, http.StatusOK, "account/management/user/list", handler.Vars{
			"Users": repo.NewBook(users, page, size, total),
		})
	}
}

func userEditGet(h *handler.Handler) http.HandlerFunc {
	h.SetViewVars("account/management/user/edit", func(r *http.Request) (handler.Vars, error) {
		userID, err := router.URLParamAs[int](r, "userID")
		if err != nil {
			return nil, errors.Tracef(httputil.ErrNotFound, err)
		}

		ctx := r.Context()

		user, err := h.Repo.Account.FindUserByID(ctx, userID)
		if err != nil {
			return nil, errors.Tracef(err)
		}

		var userRoleIDs []int
		if user.Roles != nil {
			userRoleIDs = make([]int, len(user.Roles))

			for i, role := range user.Roles {
				userRoleIDs[i] = role.ID
			}
		}

		roles, _, err := h.Repo.Account.FindRoles(ctx)

		vars := handler.Vars{
			"User":        user,
			"UserRoleIDs": userRoleIDs,
			"Roles":       roles,
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
		}
		err := httputil.DecodeForm(&input, r)
		if h.ErrorView(w, r, errors.Tracef(err), "error", nil) {
			return
		}

		userID, err := router.URLParamAs[int](r, "userID")
		if h.ErrorView(w, r, errors.Tracef(err), "error", nil) {
			return
		}

		ctx := r.Context()

		passport := h.Passport(ctx)

		user, err := h.Repo.Account.FindUserByID(ctx, userID)
		if h.ErrorView(w, r, errors.Tracef(err), "error", nil) {
			return
		}

		err = h.Account.ChangeRoles(ctx, passport, userID, input.RoleIDs...)
		if h.ErrorView(w, r, errors.Tracef(err), "account/management/user/edit", nil) {
			return
		}

		h.AddFlashf(ctx, "User %v updated successfully.", user.Email)

		h.Sessions.Set(ctx, sess.HighlightID, user.ID)

		http.Redirect(w, r, h.Path("account.management.user.list"), http.StatusSeeOther)
	}
}
