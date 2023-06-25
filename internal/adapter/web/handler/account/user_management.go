package account

import (
	"fmt"
	"net/http"
	"time"

	"github.com/polyscone/tofu/internal/adapter/web/guard"
	"github.com/polyscone/tofu/internal/adapter/web/handler"
	"github.com/polyscone/tofu/internal/adapter/web/httputil"
	"github.com/polyscone/tofu/internal/adapter/web/sess"
	"github.com/polyscone/tofu/internal/app/account"
	"github.com/polyscone/tofu/internal/pkg/http/router"
	"github.com/polyscone/tofu/internal/repository"
)

func UserManagement(h *handler.Handler, mux *router.ServeMux) {
	mux.Prefix("/users", func(mux *router.ServeMux) {
		mux.Before(h.RequireAuth(func(p guard.Passport) bool { return p.Account.CanViewUsers() }))

		mux.Get("/", userListGet(h), "account.management.user.list")

		mux.Prefix("/:userID", func(mux *router.ServeMux) {
			mux.Before(h.RequireAuth(func(p guard.Passport) bool { return p.Account.CanEditUsers() }))

			mux.Get("/", userEditGet(h), "account.management.user.edit")
			mux.Post("/", userEditPost(h), "account.management.user.edit.post")

		})

		mux.Prefix("/:userID", func(mux *router.ServeMux) {
			mux.Before(h.RequireAuth(func(p guard.Passport) bool { return p.Account.CanReviewTOTPResets() }))

			mux.Prefix("/totp-reset-review", func(mux *router.ServeMux) {
				mux.Get("/", userTOTPResetReviewGet(h), "account.management.user.totp_reset_review")

				mux.Prefix("/approve", func(mux *router.ServeMux) {
					mux.Get("/", userTOTPResetApproveGet(h), "account.management.user.totp_reset_approve")
					mux.Post("/", userTOTPResetApprovePost(h), "account.management.user.totp_reset_approve.post")
				})

				mux.Prefix("/deny", func(mux *router.ServeMux) {
					mux.Get("/", userTOTPResetDenyGet(h), "account.management.user.totp_reset_deny")
					mux.Post("/", userTOTPResetDenyPost(h), "account.management.user.totp_reset_deny.post")
				})
			})
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
			h.HTML.ErrorView(w, r, "find users page by search", err, "site/error", nil)

			return
		}

		h.HTML.View(w, r, http.StatusOK, "site/account/management/user/list", handler.Vars{
			"Users": repository.NewBook(users, page, size, total),
		})
	}
}

func userEditGet(h *handler.Handler) http.HandlerFunc {
	h.SetViewVars("site/account/management/user/edit", func(r *http.Request) (handler.Vars, error) {
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
		h.HTML.View(w, r, http.StatusOK, "site/account/management/user/edit", nil)
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
			h.HTML.ErrorView(w, r, "decode form", err, "site/error", nil)

			return
		}

		userID, err := router.URLParamAs[int](r, "userID")
		if err != nil {
			h.HTML.ErrorView(w, r, "URL param as", err, "site/error", nil)

			return
		}

		ctx := r.Context()
		passport := h.Passport(ctx)

		user, err := h.Repo.Account.FindUserByID(ctx, userID)
		if err != nil {
			h.HTML.ErrorView(w, r, "find user by id", err, "site/error", nil)

			return
		}

		err = h.Account.ChangeRoles(ctx, passport.Account, userID, input.RoleIDs, input.Grants, input.Denials)
		if err != nil {
			h.HTML.ErrorView(w, r, "change roles", err, "site/account/management/user/edit", nil)

			return
		}

		h.AddFlashf(ctx, "User %v updated successfully.", user.Email)

		h.Sessions.Set(ctx, sess.HighlightID, user.ID)

		http.Redirect(w, r, h.Path("account.management.user.list"), http.StatusSeeOther)
	}
}

func userTOTPResetReviewGet(h *handler.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, err := router.URLParamAs[int](r, "userID")
		if err != nil {
			h.HTML.ErrorView(w, r, "URL param as", err, "site/error", nil)

			return
		}

		ctx := r.Context()

		user, err := h.Repo.Account.FindUserByID(ctx, userID)
		if err != nil {
			h.HTML.ErrorView(w, r, "find user by id", err, "site/error", nil)

			return
		}

		h.HTML.View(w, r, http.StatusOK, "site/account/management/user/totp_reset_review", handler.Vars{
			"User": user,
		})
	}
}

func userTOTPResetApproveGet(h *handler.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, err := router.URLParamAs[int](r, "userID")
		if err != nil {
			h.HTML.ErrorView(w, r, "URL param as", err, "site/error", nil)

			return
		}

		ctx := r.Context()

		user, err := h.Repo.Account.FindUserByID(ctx, userID)
		if err != nil {
			h.HTML.ErrorView(w, r, "find user by id", err, "site/error", nil)

			return
		}

		h.HTML.View(w, r, http.StatusOK, "site/account/management/user/totp_reset_approve", handler.Vars{
			"User": user,
		})
	}
}

func userTOTPResetApprovePost(h *handler.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, err := router.URLParamAs[int](r, "userID")
		if err != nil {
			h.HTML.ErrorView(w, r, "URL param as", err, "site/error", nil)

			return
		}

		ctx := r.Context()
		log := h.Logger(ctx)
		config := h.Config(ctx)

		user, err := h.Repo.Account.FindUserByID(ctx, userID)
		if err != nil {
			h.HTML.ErrorView(w, r, "find user by id", err, "site/error", nil)

			return
		}

		if err := h.Account.ApproveTOTPResetRequest(ctx, user.ID); err != nil {
			h.HTML.ErrorView(w, r, "approve TOTP reset request", err, "site/error", nil)

			return
		}

		tok, err := h.Repo.Web.AddResetTOTPToken(ctx, user.Email, 48*time.Hour)
		if err != nil {
			log.Error("reset password: add reset password token", "error", err)

			return
		}

		recipients := handler.EmailRecipients{
			From: config.SystemEmail,
			To:   []string{user.Email},
		}
		vars := handler.Vars{
			"Token": tok,
		}
		if err := h.SendEmail(ctx, recipients, "totp_reset_approved", vars); err != nil {
			log.Error("reset password: send email", "error", err)
		}

		h.AddFlashf(ctx, "Two-factor authentication reset request approved for %v.", user.Email)

		http.Redirect(w, r, h.Path("account.management.user.list"), http.StatusSeeOther)
	}
}

func userTOTPResetDenyGet(h *handler.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, err := router.URLParamAs[int](r, "userID")
		if err != nil {
			h.HTML.ErrorView(w, r, "URL param as", err, "site/error", nil)

			return
		}

		ctx := r.Context()

		user, err := h.Repo.Account.FindUserByID(ctx, userID)
		if err != nil {
			h.HTML.ErrorView(w, r, "find user by id", err, "site/error", nil)

			return
		}

		h.HTML.View(w, r, http.StatusOK, "site/account/management/user/totp_reset_deny", handler.Vars{
			"User": user,
		})
	}
}

func userTOTPResetDenyPost(h *handler.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, err := router.URLParamAs[int](r, "userID")
		if err != nil {
			h.HTML.ErrorView(w, r, "URL param as", err, "site/error", nil)

			return
		}

		ctx := r.Context()

		user, err := h.Repo.Account.FindUserByID(ctx, userID)
		if err != nil {
			h.HTML.ErrorView(w, r, "find user by id", err, "site/error", nil)

			return
		}

		if err := h.Account.DenyTOTPResetRequest(ctx, user.ID); err != nil {
			h.HTML.ErrorView(w, r, "deny TOTP reset request", err, "site/error", nil)

			return
		}

		h.AddFlashf(ctx, "Two-factor authentication reset request denied for %v.", user.Email)

		http.Redirect(w, r, h.Path("account.management.user.list"), http.StatusSeeOther)
	}
}
