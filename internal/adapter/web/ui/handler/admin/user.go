package admin

import (
	"net/http"

	"github.com/polyscone/tofu/internal/adapter/web/handler"
	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/http/router"
	"github.com/polyscone/tofu/internal/repo"
)

func Users(svc *handler.Services, mux *router.ServeMux) {
	mux.Prefix("/users", func(mux *router.ServeMux) {
		mux.Get("/", userListGet(svc), "admin.user.list")
		mux.Get("/:userID", userEditGet(svc), "admin.user.edit")
	})
}

func userListGet(svc *handler.Services) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		search := r.URL.Query().Get("search")
		page, size := svc.Pagination(r)
		users, total, err := svc.Repo.Account.FindUsersByPage(ctx, search, page, size)
		if svc.ErrorView(w, r, errors.Tracef(err), "error", nil) {
			return
		}

		svc.View(w, r, http.StatusOK, "admin/user/list", handler.Vars{
			"Users": repo.NewBook(users, page, size, total),
		})
	}
}

func userEditGet(svc *handler.Services) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, err := router.URLParamAs[int](r, "userID")
		if svc.ErrorView(w, r, errors.Tracef(err), "error", nil) {
			return
		}

		ctx := r.Context()

		user, err := svc.Repo.Account.FindUserByID(ctx, userID)
		if svc.ErrorView(w, r, errors.Tracef(err), "error", nil) {
			return
		}

		svc.View(w, r, http.StatusOK, "admin/user/edit", handler.Vars{
			"User": user,
		})
	}
}
