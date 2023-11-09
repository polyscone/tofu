package account

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/polyscone/tofu/internal/adapter/web/guard"
	"github.com/polyscone/tofu/internal/adapter/web/handler"
	"github.com/polyscone/tofu/internal/adapter/web/httputil"
	"github.com/polyscone/tofu/internal/adapter/web/sess"
	"github.com/polyscone/tofu/internal/adapter/web/ui"
	"github.com/polyscone/tofu/internal/app/account"
	"github.com/polyscone/tofu/internal/pkg/http/router"
	"github.com/polyscone/tofu/internal/repository"
)

func UserManagement(h *ui.Handler, mux *router.ServeMux) {
	mux.Prefix("/users", func(mux *router.ServeMux) {
		mux.Before(h.CanAccess(func(p guard.Passport) bool { return p.Account.CanViewUsers() }))

		mux.Get("/", userListGet(h), "account.management.user.list")

		mux.Prefix("/new", func(mux *router.ServeMux) {
			mux.Before(h.CanAccess(func(p guard.Passport) bool { return p.Account.CanInviteUsers() }))

			mux.Get("/", userNewGet(h), "account.management.user.new")
			mux.Post("/", userNewPost(h), "account.management.user.new.post")
		})

		mux.Prefix("/{userID}", func(mux *router.ServeMux) {
			mux.Before(func(next http.HandlerFunc) http.HandlerFunc {
				return func(w http.ResponseWriter, r *http.Request) {
					userID, ok := router.URLParamAs[int](r, "userID")
					if !ok {
						next(w, r)

						return
					}

					canAccess := h.CanAccess(func(p guard.Passport) bool {
						return p.Account.CanChangeRoles(userID) || p.Account.CanActivateUsers()
					})

					canAccess(next)(w, r)
				}
			})

			mux.Get("/", userEditGet(h), "account.management.user.edit")
			mux.Post("/roles", userEditRolesPost(h), "account.management.user.roles.post")
			mux.Post("/suspend", userSuspendPost(h), "account.management.user.suspend.post")
			mux.Post("/unsuspend", userUnsuspendPost(h), "account.management.user.unsuspend.post")

			mux.Prefix("/activate", func(mux *router.ServeMux) {
				mux.Get("/", userActivateGet(h), "account.management.user.activate")
				mux.Post("/", userActivatePost(h), "account.management.user.activate.post")
			})

			mux.Prefix("/totp-reset-review", func(mux *router.ServeMux) {
				mux.Before(h.CanAccess(func(p guard.Passport) bool { return p.Account.CanReviewTOTPResets() }))

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

func userListGet(h *ui.Handler) http.HandlerFunc {
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

func userNewGet(h *ui.Handler) http.HandlerFunc {
	h.SetViewVars("site/account/management/user/new", func(r *http.Request) (handler.Vars, error) {
		ctx := r.Context()

		roles, _, err := h.Repo.Account.FindRoles(ctx, account.SuperRole.ID)
		if err != nil {
			return nil, fmt.Errorf("find roles: %w", err)
		}

		vars := handler.Vars{
			"Roles":            roles,
			"SuperRole":        account.SuperRole,
			"PermissionGroups": guard.PermissionGroups,
		}

		return vars, nil
	})

	return func(w http.ResponseWriter, r *http.Request) {
		h.HTML.View(w, r, http.StatusOK, "site/account/management/user/new", nil)
	}
}

func userNewPost(h *ui.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Email string `form:"email"`
		}
		if err := httputil.DecodeRequestForm(&input, r); err != nil {
			h.HTML.ErrorView(w, r, "decode form", err, "site/error", nil)

			return
		}

		ctx := r.Context()
		passport := h.Passport(ctx)

		user, err := h.Svc.Account.InviteUser(ctx, passport.Account, input.Email)
		if err != nil {
			h.HTML.ErrorView(w, r, "invite", err, "site/account/management/user/new", nil)

			return
		}

		h.AddFlashf(ctx, "An invite to verify an account has been sent to %q.", user.Email)

		h.Sessions.Set(ctx, sess.SortTopID, user.ID)
		h.Sessions.Set(ctx, sess.HighlightID, user.ID)

		http.Redirect(w, r, h.Path("account.management.user.edit", "{userID}", user.ID), http.StatusSeeOther)
	}
}

func userEditGet(h *ui.Handler) http.HandlerFunc {
	h.SetViewVars("site/account/management/user/edit", func(r *http.Request) (handler.Vars, error) {
		userID, ok := router.URLParamAs[int](r, "userID")
		if !ok {
			return nil, errors.New("URL param as: invalid int")
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
		if err != nil {
			return nil, fmt.Errorf("find roles: %w", err)
		}

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

func userEditRolesPost(h *ui.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			RoleIDs []int    `form:"roles"`
			Grants  []string `form:"grants"`
			Denials []string `form:"denials"`
		}
		if err := httputil.DecodeRequestForm(&input, r); err != nil {
			h.HTML.ErrorView(w, r, "decode form", err, "site/error", nil)

			return
		}

		userID, ok := router.URLParamAs[int](r, "userID")
		if !ok {
			h.HTML.ErrorView(w, r, "URL param as", errors.New("invalid int"), "site/error", nil)

			return
		}

		ctx := r.Context()
		passport := h.Passport(ctx)

		user, err := h.Repo.Account.FindUserByID(ctx, userID)
		if err != nil {
			h.HTML.ErrorView(w, r, "find user by id", err, "site/error", nil)

			return
		}

		err = h.Svc.Account.ChangeRoles(ctx, passport.Account, userID, input.RoleIDs, input.Grants, input.Denials)
		if err != nil {
			h.HTML.ErrorView(w, r, "change roles", err, "site/account/management/user/edit", nil)

			return
		}

		h.AddFlashf(ctx, "User %v updated successfully.", user.Email)

		h.Sessions.Set(ctx, sess.HighlightID, user.ID)

		http.Redirect(w, r, h.PathQuery(r, "account.management.user.list"), http.StatusSeeOther)
	}
}

func userSuspendPost(h *ui.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			SuspendedReason string `form:"suspended-reason"`
		}
		if err := httputil.DecodeRequestForm(&input, r); err != nil {
			h.HTML.ErrorView(w, r, "decode form", err, "site/error", nil)

			return
		}

		userID, ok := router.URLParamAs[int](r, "userID")
		if !ok {
			h.HTML.ErrorView(w, r, "URL param as", errors.New("invalid int"), "site/error", nil)

			return
		}

		ctx := r.Context()
		passport := h.Passport(ctx)

		user, err := h.Repo.Account.FindUserByID(ctx, userID)
		if err != nil {
			h.HTML.ErrorView(w, r, "find user by id", err, "site/error", nil)

			return
		}

		wasSuspended := user.IsSuspended()

		err = h.Svc.Account.SuspendUser(ctx, passport.Account, userID, input.SuspendedReason)
		if err != nil {
			h.HTML.ErrorView(w, r, "suspend user", err, "site/account/management/user/edit", nil)

			return
		}

		if wasSuspended {
			h.AddFlashf(ctx, "Updated suspended reason for %v.", user.Email)
		} else {
			h.AddFlashf(ctx, "User %v was suspended.", user.Email)
		}

		h.Sessions.Set(ctx, sess.HighlightID, user.ID)

		http.Redirect(w, r, h.PathQuery(r, "account.management.user.list"), http.StatusSeeOther)
	}
}

func userUnsuspendPost(h *ui.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := router.URLParamAs[int](r, "userID")
		if !ok {
			h.HTML.ErrorView(w, r, "URL param as", errors.New("invalid int"), "site/error", nil)

			return
		}

		ctx := r.Context()
		passport := h.Passport(ctx)

		user, err := h.Repo.Account.FindUserByID(ctx, userID)
		if err != nil {
			h.HTML.ErrorView(w, r, "find user by id", err, "site/error", nil)

			return
		}

		err = h.Svc.Account.UnsuspendUser(ctx, passport.Account, userID)
		if err != nil {
			h.HTML.ErrorView(w, r, "unsuspend user", err, "site/account/management/user/edit", nil)

			return
		}

		h.AddFlashf(ctx, "User %v was unsuspended.", user.Email)

		h.Sessions.Set(ctx, sess.HighlightID, user.ID)

		http.Redirect(w, r, h.PathQuery(r, "account.management.user.list"), http.StatusSeeOther)
	}
}

func userActivateGet(h *ui.Handler) http.HandlerFunc {
	h.SetViewVars("site/account/management/user/activate", func(r *http.Request) (handler.Vars, error) {
		userID, ok := router.URLParamAs[int](r, "userID")
		if !ok {
			return nil, errors.New("URL param as: invalid int")
		}

		ctx := r.Context()

		user, err := h.Repo.Account.FindUserByID(ctx, userID)
		if err != nil {
			return nil, fmt.Errorf("find user by id: %w", err)
		}

		vars := handler.Vars{"User": user}

		return vars, nil
	})

	return func(w http.ResponseWriter, r *http.Request) {
		h.HTML.View(w, r, http.StatusOK, "site/account/management/user/activate", nil)
	}
}

func userActivatePost(h *ui.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := router.URLParamAs[int](r, "userID")
		if !ok {
			h.HTML.ErrorView(w, r, "URL param as", errors.New("invalid int"), "site/error", nil)

			return
		}

		ctx := r.Context()
		passport := h.Passport(ctx)

		user, err := h.Repo.Account.FindUserByID(ctx, userID)
		if err != nil {
			h.HTML.ErrorView(w, r, "find user by id", err, "site/error", nil)

			return
		}

		err = h.Svc.Account.ActivateUser(ctx, passport.Account, userID)
		if err != nil {
			h.HTML.ErrorView(w, r, "activate user", err, "site/account/management/user/activate", nil)

			return
		}

		h.AddFlashf(ctx, "User %v activated successfully.", user.Email)

		h.Sessions.Set(ctx, sess.HighlightID, user.ID)

		http.Redirect(w, r, h.PathQuery(r, "account.management.user.list"), http.StatusSeeOther)
	}
}

func userTOTPResetReviewGet(h *ui.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := router.URLParamAs[int](r, "userID")
		if !ok {
			h.HTML.ErrorView(w, r, "URL param as", errors.New("invalid int"), "site/error", nil)

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

func userTOTPResetApproveGet(h *ui.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := router.URLParamAs[int](r, "userID")
		if !ok {
			h.HTML.ErrorView(w, r, "URL param as", errors.New("invalid int"), "site/error", nil)

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

func userTOTPResetApprovePost(h *ui.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := router.URLParamAs[int](r, "userID")
		if !ok {
			h.HTML.ErrorView(w, r, "URL param as", errors.New("invalid int"), "site/error", nil)

			return
		}

		ctx := r.Context()
		logger := h.Logger(ctx)
		config := h.Config(ctx)

		user, err := h.Repo.Account.FindUserByID(ctx, userID)
		if err != nil {
			h.HTML.ErrorView(w, r, "find user by id", err, "site/error", nil)

			return
		}

		if err := h.Svc.Account.ApproveTOTPResetRequest(ctx, user.ID); err != nil {
			h.HTML.ErrorView(w, r, "approve TOTP reset request", err, "site/error", nil)

			return
		}

		tok, err := h.Repo.Web.AddResetTOTPToken(ctx, user.Email, 48*time.Hour)
		if err != nil {
			logger.Error("reset TOTP: add reset TOTP token", "error", err)

			return
		}

		vars := handler.Vars{"Token": tok}
		if err := h.SendEmail(ctx, config.SystemEmail, user.Email, "totp_reset_approved", vars); err != nil {
			logger.Error("reset TOTP: send email", "error", err)
		}

		h.AddFlashf(ctx, "Two-factor authentication reset request approved for %v.", user.Email)

		http.Redirect(w, r, h.Path("account.management.user.list"), http.StatusSeeOther)
	}
}

func userTOTPResetDenyGet(h *ui.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := router.URLParamAs[int](r, "userID")
		if !ok {
			h.HTML.ErrorView(w, r, "URL param as", errors.New("invalid int"), "site/error", nil)

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

func userTOTPResetDenyPost(h *ui.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := router.URLParamAs[int](r, "userID")
		if !ok {
			h.HTML.ErrorView(w, r, "URL param as", errors.New("invalid int"), "site/error", nil)

			return
		}

		ctx := r.Context()

		user, err := h.Repo.Account.FindUserByID(ctx, userID)
		if err != nil {
			h.HTML.ErrorView(w, r, "find user by id", err, "site/error", nil)

			return
		}

		if err := h.Svc.Account.DenyTOTPResetRequest(ctx, user.ID); err != nil {
			h.HTML.ErrorView(w, r, "deny TOTP reset request", err, "site/error", nil)

			return
		}

		h.AddFlashf(ctx, "Two-factor authentication reset request denied for %v.", user.Email)

		http.Redirect(w, r, h.Path("account.management.user.list"), http.StatusSeeOther)
	}
}
