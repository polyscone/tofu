package account

import (
	"net/http"

	"github.com/polyscone/tofu/internal/adapter/web/handler"
	"github.com/polyscone/tofu/internal/adapter/web/httputil"
	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/http/router"
	"github.com/polyscone/tofu/internal/repo"
)

func UserManagement(svc *handler.Services, mux *router.ServeMux) {
	mux.Prefix("/users", func(mux *router.ServeMux) {
		mux.Get("/", userListGet(svc), "account.management.user.list")

		mux.Prefix("/:userID", func(mux *router.ServeMux) {
			mux.Get("/", userEditGet(svc), "account.management.user.edit")
			mux.Post("/", userEditPost(svc), "account.management.user.edit.post")
		})
	})
}

func userListGet(svc *handler.Services) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		search := r.URL.Query().Get("search")
		page, size := svc.Pagination(r)
		users, total, err := svc.Repo.Account.FindUsersPageBySearch(ctx, search, page, size)
		if svc.ErrorView(w, r, errors.Tracef(err), "error", nil) {
			return
		}

		svc.View(w, r, http.StatusOK, "account/management/user/list", handler.Vars{
			"Users": repo.NewBook(users, page, size, total),
		})
	}
}

func userEditGet(svc *handler.Services) http.HandlerFunc {
	svc.SetViewVars("account/management/user/edit", func(r *http.Request) (handler.Vars, error) {
		userID, err := router.URLParamAs[int](r, "userID")
		if err != nil {
			return nil, errors.Tracef(err, httputil.ErrNotFound)
		}

		ctx := r.Context()

		user, err := svc.Repo.Account.FindUserByID(ctx, userID)
		if err != nil {
			return nil, errors.Tracef(err)
		}

		vars := handler.Vars{
			"User": user,
		}

		return vars, nil
	})

	return func(w http.ResponseWriter, r *http.Request) {
		svc.View(w, r, http.StatusOK, "account/management/user/edit", nil)
	}
}

func userEditPost(svc *handler.Services) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		svc.View(w, r, http.StatusOK, "account/management/user/edit", nil)
	}
}
