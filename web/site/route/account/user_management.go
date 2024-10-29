package account

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/polyscone/tofu/app"
	"github.com/polyscone/tofu/internal/collection"
	"github.com/polyscone/tofu/internal/httpx"
	"github.com/polyscone/tofu/internal/httpx/router"
	"github.com/polyscone/tofu/internal/i18n"
	"github.com/polyscone/tofu/web/guard"
	"github.com/polyscone/tofu/web/handler"
	"github.com/polyscone/tofu/web/site/ui"
)

func RegisterUserManagementHandlers(h *ui.Handler, mux *router.ServeMux) {
	mux.Group(func(mux *router.ServeMux) {
		mux.Before(h.RequireSignIn)

		mux.Group(func(mux *router.ServeMux) {
			mux.Before(h.CanAccess(func(p guard.Passport) bool { return p.Account.CanViewUsers() }))

			mux.HandleFunc("GET /admin/account/users", userListGet(h), "account.management.user.list")

		})

		mux.Group(func(mux *router.ServeMux) {
			mux.Before(h.CanAccess(func(p guard.Passport) bool { return p.Account.CanInviteUsers() }))

			mux.HandleFunc("GET /admin/account/users/new", userNewGet(h), "account.management.user.new")
			mux.HandleFunc("POST /admin/account/users/new", userNewPost(h), "account.management.user.new.post")
		})

		mux.Group(func(mux *router.ServeMux) {
			mux.Before(func(next http.HandlerFunc) http.HandlerFunc {
				return func(w http.ResponseWriter, r *http.Request) {
					userID, _ := strconv.Atoi(r.PathValue("userID"))
					canAccess := h.CanAccess(func(p guard.Passport) bool {
						return p.Account.CanChangeRoles(userID) ||
							p.Account.CanSuspendUsers()
					})

					canAccess(next)(w, r)
				}
			})

			mux.HandleFunc("GET /admin/account/users/{userID}", userEditGet(h), "account.management.user.edit")
			mux.HandleFunc("POST /admin/account/users/{userID}/roles", userEditRolesPost(h), "account.management.user.roles.post")
			mux.HandleFunc("POST /admin/account/users/{userID}/suspend", userSuspendPost(h), "account.management.user.suspend.post")
			mux.HandleFunc("POST /admin/account/users/{userID}/unsuspend", userUnsuspendPost(h), "account.management.user.unsuspend.post")
		})

		mux.Group(func(mux *router.ServeMux) {
			mux.Before(h.CanAccess(func(p guard.Passport) bool { return p.Account.CanActivateUsers() }))

			mux.HandleFunc("GET /admin/account/users/{userID}/activate", userActivateGet(h), "account.management.user.activate")
			mux.HandleFunc("POST /admin/account/users/{userID}/activate", userActivatePost(h), "account.management.user.activate.post")
		})

		mux.Group(func(mux *router.ServeMux) {
			mux.Before(h.CanAccess(func(p guard.Passport) bool { return p.Account.CanReviewTOTPResets() }))

			mux.HandleFunc("GET /admin/account/users/{userID}/totp-reset-review", userTOTPResetReviewGet(h), "account.management.user.totp_reset_review")

			mux.HandleFunc("GET /admin/account/users/{userID}/totp-reset-review/approve", userTOTPResetApproveGet(h), "account.management.user.totp_reset_approve")
			mux.HandleFunc("POST /admin/account/users/{userID}/totp-reset-review/approve", userTOTPResetApprovePost(h), "account.management.user.totp_reset_approve.post")

			mux.HandleFunc("GET /admin/account/users/{userID}/totp-reset-review/deny", userTOTPResetDenyGet(h), "account.management.user.totp_reset_deny")
			mux.HandleFunc("POST /admin/account/users/{userID}/totp-reset-review/deny", userTOTPResetDenyPost(h), "account.management.user.totp_reset_deny.post")
		})
	})
}

func userListGet(h *ui.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		sortTopID := h.Session.PopSortTopID(ctx)
		if sortTopID == 0 {
			sortTopID = h.User(ctx).ID
		}
		sorts := r.URL.Query()["sort"]
		search := r.URL.Query().Get("search")
		page, size := httpx.Pagination(r)
		users, total, err := h.Repo.Account.FindUsersPageBySearch(ctx, page, size, sortTopID, sorts, search)
		if err != nil {
			h.HTML.ErrorView(w, r, "find users page by search", err, "error", nil)

			return
		}

		h.HTML.View(w, r, http.StatusOK, "account/management/user/list", handler.Vars{
			"Users": collection.NewBook(users, page, size, total),
		})
	}
}

func userNewGet(h *ui.Handler) http.HandlerFunc {
	h.HTML.SetViewVars("account/management/user/new", func(r *http.Request) (handler.Vars, error) {
		ctx := r.Context()

		roles, _, err := h.Repo.Account.FindRoles(ctx, h.SuperRole.ID)
		if err != nil {
			return nil, fmt.Errorf("find roles: %w", err)
		}

		vars := handler.Vars{
			"Roles":            roles,
			"SuperRole":        h.SuperRole,
			"PermissionGroups": guard.PermissionGroups,
		}

		return vars, nil
	})

	return func(w http.ResponseWriter, r *http.Request) {
		h.HTML.View(w, r, http.StatusOK, "account/management/user/new", nil)
	}
}

func userNewPost(h *ui.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Email string `form:"email"`
		}
		if err := httpx.DecodeRequestForm(&input, r); err != nil {
			h.HTML.ErrorView(w, r, "decode form", err, "error", nil)

			return
		}

		ctx := r.Context()
		passport := h.Passport(ctx)

		user, err := h.Svc.Account.InviteUser(ctx, passport.Account, input.Email)
		if err != nil {
			h.HTML.ErrorView(w, r, "invite", err, "account/management/user/new", nil)

			return
		}

		h.AddFlashf(ctx, i18n.M("site.account.user_management.flash.invite_sent", "email", user.Email))

		h.Session.SetSortTopID(ctx, user.ID)
		h.Session.SetHighlightID(ctx, user.ID)

		http.Redirect(w, r, h.Path("account.management.user.edit", "{userID}", user.ID), http.StatusSeeOther)
	}
}

func userEditGet(h *ui.Handler) http.HandlerFunc {
	h.HTML.SetViewVars("account/management/user/edit", func(r *http.Request) (handler.Vars, error) {
		ctx := r.Context()

		userID, _ := strconv.Atoi(r.PathValue("userID"))
		user, err := h.Repo.Account.FindUserByID(ctx, userID)
		if err != nil {
			return nil, fmt.Errorf("find user by id: %w", err)
		}

		userIsSuper := h.PassportByUser(ctx, user).IsSuper

		var userRoleIDs []int
		if user.Roles != nil {
			userRoleIDs = make([]int, len(user.Roles))

			for i, role := range user.Roles {
				userRoleIDs[i] = role.ID
			}
		}

		roles, _, err := h.Repo.Account.FindRoles(ctx, h.SuperRole.ID)
		if err != nil {
			return nil, fmt.Errorf("find roles: %w", err)
		}

		vars := handler.Vars{
			"User":             user,
			"UserIsSuper":      userIsSuper,
			"UserRoleIDs":      userRoleIDs,
			"Roles":            roles,
			"SuperRole":        h.SuperRole,
			"PermissionGroups": guard.PermissionGroups,
		}

		return vars, nil
	})

	return func(w http.ResponseWriter, r *http.Request) {
		h.HTML.View(w, r, http.StatusOK, "account/management/user/edit", nil)
	}
}

func userEditRolesPost(h *ui.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			RoleIDs []int    `form:"roles"`
			Grants  []string `form:"grants"`
			Denials []string `form:"denials"`
		}
		if err := httpx.DecodeRequestForm(&input, r); err != nil {
			h.HTML.ErrorView(w, r, "decode form", err, "error", nil)

			return
		}

		ctx := r.Context()
		passport := h.Passport(ctx)
		userID, _ := strconv.Atoi(r.PathValue("userID"))

		var containsSuper bool
		for _, roleID := range input.RoleIDs {
			if roleID == h.SuperRole.ID {
				containsSuper = true

				if !passport.Account.CanAssignSuperRole(userID) {
					h.HTML.ErrorView(w, r, "check role ids", app.ErrForbidden, "account/management/user/edit", nil)

					return
				}
			}
		}

		user, err := h.Repo.Account.FindUserByID(ctx, userID)
		if err != nil {
			h.HTML.ErrorView(w, r, "find user by id", err, "error", nil)

			return
		}

		if p := h.PassportByUser(ctx, user); p.IsSuper && !containsSuper {
			h.HTML.ErrorView(w, r, "cannot remove super role", app.ErrForbidden, "account/management/user/edit", nil)

			return
		}

		_, err = h.Svc.Account.ChangeRoles(ctx, passport.Account, userID, input.RoleIDs, input.Grants, input.Denials)
		if err != nil {
			h.HTML.ErrorView(w, r, "change roles", err, "account/management/user/edit", nil)

			return
		}

		h.AddFlashf(ctx, i18n.M("site.account.user_management.flash.updated", "email", user.Email))

		h.Session.SetHighlightID(ctx, user.ID)

		http.Redirect(w, r, h.PathQuery(r, "account.management.user.list"), http.StatusSeeOther)
	}
}

func userSuspendPost(h *ui.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			SuspendedReason string `form:"suspended-reason"`
		}
		if err := httpx.DecodeRequestForm(&input, r); err != nil {
			h.HTML.ErrorView(w, r, "decode form", err, "error", nil)

			return
		}

		ctx := r.Context()
		passport := h.Passport(ctx)

		userID, _ := strconv.Atoi(r.PathValue("userID"))
		user, err := h.Repo.Account.FindUserByID(ctx, userID)
		if err != nil {
			h.HTML.ErrorView(w, r, "find user by id", err, "error", nil)

			return
		}

		if p := h.PassportByUser(ctx, user); p.IsSuper {
			h.HTML.ErrorView(w, r, "cannot suspend a user with the super role", app.ErrForbidden, "account/management/user/edit", nil)

			return
		}

		wasSuspended := user.IsSuspended()

		_, err = h.Svc.Account.SuspendUser(ctx, passport.Account, userID, input.SuspendedReason)
		if err != nil {
			h.HTML.ErrorView(w, r, "suspend user", err, "account/management/user/edit", nil)

			return
		}

		if wasSuspended {
			h.AddFlashf(ctx, i18n.M("site.account.user_management.flash.suspended_reason_updated", "email", user.Email))
		} else {
			h.AddFlashf(ctx, i18n.M("site.account.user_management.flash.suspended", "email", user.Email))
		}

		h.Session.SetHighlightID(ctx, user.ID)

		q := r.URL.Query()

		q.Del("suspend")

		r.URL.RawQuery = q.Encode()

		http.Redirect(w, r, h.PathQuery(r, "account.management.user.list"), http.StatusSeeOther)
	}
}

func userUnsuspendPost(h *ui.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		passport := h.Passport(ctx)

		userID, _ := strconv.Atoi(r.PathValue("userID"))
		user, err := h.Svc.Account.UnsuspendUser(ctx, passport.Account, userID)
		if err != nil {
			h.HTML.ErrorView(w, r, "unsuspend user", err, "account/management/user/edit", nil)

			return
		}

		h.AddFlashf(ctx, i18n.M("site.account.user_management.flash.unsuspended", "email", user.Email))

		h.Session.SetHighlightID(ctx, user.ID)

		http.Redirect(w, r, h.PathQuery(r, "account.management.user.list"), http.StatusSeeOther)
	}
}

func userActivateGet(h *ui.Handler) http.HandlerFunc {
	h.HTML.SetViewVars("account/management/user/activate", func(r *http.Request) (handler.Vars, error) {
		ctx := r.Context()

		userID, _ := strconv.Atoi(r.PathValue("userID"))
		user, err := h.Repo.Account.FindUserByID(ctx, userID)
		if err != nil {
			return nil, fmt.Errorf("find user by id: %w", err)
		}

		vars := handler.Vars{"User": user}

		return vars, nil
	})

	return func(w http.ResponseWriter, r *http.Request) {
		h.HTML.View(w, r, http.StatusOK, "account/management/user/activate", nil)
	}
}

func userActivatePost(h *ui.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		passport := h.Passport(ctx)

		userID, _ := strconv.Atoi(r.PathValue("userID"))
		user, err := h.Svc.Account.ActivateUser(ctx, passport.Account, userID)
		if err != nil {
			h.HTML.ErrorView(w, r, "activate user", err, "account/management/user/activate", nil)

			return
		}

		h.AddFlashf(ctx, i18n.M("site.account.user_management.flash.activated", "email", user.Email))

		h.Session.SetHighlightID(ctx, user.ID)

		http.Redirect(w, r, h.PathQuery(r, "account.management.user.list"), http.StatusSeeOther)
	}
}

func userTOTPResetReviewGet(h *ui.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		userID, _ := strconv.Atoi(r.PathValue("userID"))
		user, err := h.Repo.Account.FindUserByID(ctx, userID)
		if err != nil {
			h.HTML.ErrorView(w, r, "find user by id", err, "error", nil)

			return
		}

		h.HTML.View(w, r, http.StatusOK, "account/management/user/totp_reset_review", handler.Vars{
			"User": user,
		})
	}
}

func userTOTPResetApproveGet(h *ui.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		userID, _ := strconv.Atoi(r.PathValue("userID"))
		user, err := h.Repo.Account.FindUserByID(ctx, userID)
		if err != nil {
			h.HTML.ErrorView(w, r, "find user by id", err, "error", nil)

			return
		}

		h.HTML.View(w, r, http.StatusOK, "account/management/user/totp_reset_approve", handler.Vars{
			"User": user,
		})
	}
}

func userTOTPResetApprovePost(h *ui.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		logger := h.Logger(ctx)
		config := h.Config(ctx)

		userID, _ := strconv.Atoi(r.PathValue("userID"))
		user, err := h.Svc.Account.ApproveTOTPResetRequest(ctx, userID)
		if err != nil {
			h.HTML.ErrorView(w, r, "approve TOTP reset request", err, "error", nil)

			return
		}

		tok, err := h.Repo.Web.AddResetTOTPToken(ctx, user.Email, 48*time.Hour)
		if err != nil {
			logger.Error("reset TOTP: add reset TOTP token", "error", err)

			return
		}

		vars := handler.Vars{
			"Token":    tok,
			"ResetURL": fmt.Sprintf("%v://%v%v?token=%v", h.Scheme, h.Host, h.Path("account.totp.reset"), tok),
		}
		if err := h.SendEmail(ctx, config.SystemEmail, user.Email, "totp_reset_approved", vars); err != nil {
			logger.Error("reset TOTP: send email", "error", err)
		}

		h.AddFlashf(ctx, i18n.M("site.account.user_management.flash.totp_reset_request_approved", "email", user.Email))

		http.Redirect(w, r, h.Path("account.management.user.list"), http.StatusSeeOther)
	}
}

func userTOTPResetDenyGet(h *ui.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		userID, _ := strconv.Atoi(r.PathValue("userID"))
		user, err := h.Repo.Account.FindUserByID(ctx, userID)
		if err != nil {
			h.HTML.ErrorView(w, r, "find user by id", err, "error", nil)

			return
		}

		h.HTML.View(w, r, http.StatusOK, "account/management/user/totp_reset_deny", handler.Vars{
			"User": user,
		})
	}
}

func userTOTPResetDenyPost(h *ui.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		userID, _ := strconv.Atoi(r.PathValue("userID"))
		user, err := h.Svc.Account.DenyTOTPResetRequest(ctx, userID)
		if err != nil {
			h.HTML.ErrorView(w, r, "deny TOTP reset request", err, "error", nil)

			return
		}

		h.AddFlashf(ctx, i18n.M("site.account.user_management.flash.totp_reset_request_denied", "email", user.Email))

		http.Redirect(w, r, h.Path("account.management.user.list"), http.StatusSeeOther)
	}
}
