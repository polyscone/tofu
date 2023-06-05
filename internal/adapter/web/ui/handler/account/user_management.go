package account

import (
	"net/http"

	"github.com/polyscone/tofu/internal/adapter/web/httputil"
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

		search := r.URL.Query().Get("search")
		page, size := httputil.Pagination(r)
		users, total, err := h.Repo.Account.FindUsersPageBySearch(ctx, search, page, size)
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
			return nil, errors.Tracef(err, httputil.ErrNotFound)
		}

		ctx := r.Context()

		user, err := h.Repo.Account.FindUserByID(ctx, userID)
		if err != nil {
			return nil, errors.Tracef(err)
		}

		vars := handler.Vars{
			"User": user,
		}

		return vars, nil
	})

	return func(w http.ResponseWriter, r *http.Request) {
		h.View(w, r, http.StatusOK, "account/management/user/edit", nil)
	}
}

func userEditPost(h *handler.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h.View(w, r, http.StatusOK, "account/management/user/edit", nil)
	}
}
