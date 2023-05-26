package user

import (
	"net/http"

	"github.com/polyscone/tofu/internal/adapter/web/handler"
	"github.com/polyscone/tofu/internal/adapter/web/httputil"
	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/http/router"
	"github.com/polyscone/tofu/internal/repo"
)

func List(svc *handler.Services, mux *router.ServeMux) {
	mux.Get("/list", listGet(svc), "admin.user.list")

	svc.SetViewVars("admin/user/list", handler.Vars{
		"User":  nil,
		"Users": nil,
	})
}

func listGet(svc *handler.Services) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			EditID int `query:"edit"`
		}
		err := httputil.DecodeQuery(&input, r)
		if svc.ErrorView(w, r, errors.Tracef(err), "error", nil) {
			return
		}

		ctx := r.Context()

		if input.EditID > 0 {
			user, err := svc.Repo.Account.FindUserByID(ctx, input.EditID)
			if svc.ErrorView(w, r, errors.Tracef(err), "error", nil) {
				return
			}

			svc.View(w, r, http.StatusOK, "admin/user/list", handler.Vars{
				"User": user,
			})
		} else {
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
}
