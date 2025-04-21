package account

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/polyscone/tofu/app"
	"github.com/polyscone/tofu/app/account"
	"github.com/polyscone/tofu/internal/background"
	"github.com/polyscone/tofu/internal/collection"
	"github.com/polyscone/tofu/internal/httpx"
	"github.com/polyscone/tofu/internal/httpx/router"
	"github.com/polyscone/tofu/internal/i18n"
	"github.com/polyscone/tofu/web/auth"
	"github.com/polyscone/tofu/web/guard"
	"github.com/polyscone/tofu/web/handler"
	"github.com/polyscone/tofu/web/site/ui"
)

func RegisterUserManagementHandlers(h *ui.Handler, mux *router.ServeMux) {
	mux.HandleFunc("GET /admin/account/users", userListGet(h), "account.management.user.list")

	mux.HandleFunc("GET /admin/account/users/new", userNewGet(h), "account.management.user.new")
	mux.HandleFunc("POST /admin/account/users/new", userNewPost(h), "account.management.user.new.post")

	mux.HandleFunc("GET /admin/account/users/{userID}", userEditGet(h), "account.management.user.edit")

	mux.HandleFunc("GET /admin/account/users/{userID}/roles", userEditRolesGet(h))
	mux.HandleFunc("POST /admin/account/users/{userID}/roles", userEditRolesPost(h), "account.management.user.roles.post")

	mux.HandleFunc("GET /admin/account/users/{userID}/suspend", userSuspendGet(h))
	mux.HandleFunc("POST /admin/account/users/{userID}/suspend", userSuspendPost(h), "account.management.user.suspend.post")

	mux.HandleFunc("GET /admin/account/users/{userID}/unsuspend", userUnsuspendGet(h))
	mux.HandleFunc("POST /admin/account/users/{userID}/unsuspend", userUnsuspendPost(h), "account.management.user.unsuspend.post")

	mux.HandleFunc("GET /admin/account/users/{userID}/activate", userActivateGet(h), "account.management.user.activate")
	mux.HandleFunc("POST /admin/account/users/{userID}/activate", userActivatePost(h), "account.management.user.activate.post")

	mux.HandleFunc("POST /admin/account/users/{userID}/impersonate/start", userStartImpersonatePost(h), "account.management.user.impersonate.start.post")
	mux.HandleFunc("POST /admin/account/users/impersonate/stop", userStopImpersonatePost(h), "account.management.user.impersonate.stop.post")

	mux.HandleFunc("GET /admin/account/users/{userID}/totp-reset-review", userTOTPResetReviewGet(h), "account.management.user.totp_reset_review")

	mux.HandleFunc("GET /admin/account/users/{userID}/totp-reset-review/approve", userTOTPResetApproveGet(h), "account.management.user.totp_reset_approve")
	mux.HandleFunc("POST /admin/account/users/{userID}/totp-reset-review/approve", userTOTPResetApprovePost(h), "account.management.user.totp_reset_approve.post")

	mux.HandleFunc("GET /admin/account/users/{userID}/totp-reset-review/deny", userTOTPResetDenyGet(h), "account.management.user.totp_reset_deny")
	mux.HandleFunc("POST /admin/account/users/{userID}/totp-reset-review/deny", userTOTPResetDenyPost(h), "account.management.user.totp_reset_deny.post")
}

func userListGet(h *ui.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		allowed := func(p guard.Passport) bool { return p.Account.CanViewUsers() }
		if h.RequireSignIn(w, r) || h.Forbidden(w, r, allowed) {
			return
		}

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
	const view = "account/management/user/new"
	h.HTML.SetViewVars(view, func(r *http.Request) (handler.Vars, error) {
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
		allowed := func(p guard.Passport) bool { return p.Account.CanInviteUsers() }
		if h.RequireSignIn(w, r) || h.Forbidden(w, r, allowed) {
			return
		}

		h.HTML.View(w, r, http.StatusOK, view, nil)
	}
}

func userNewPost(h *ui.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		allowed := func(p guard.Passport) bool { return p.Account.CanInviteUsers() }
		if h.RequireSignIn(w, r) || h.Forbidden(w, r, allowed) {
			return
		}

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
			h.HTML.ErrorView(w, r, "invite", err, h.Session.LastView(ctx), nil)

			return
		}

		h.AddFlashf(ctx, i18n.M("site.account.user_management.flash.invite_sent", "email", user.Email))

		h.Session.SetSortTopID(ctx, user.ID)
		h.Session.SetHighlightID(ctx, user.ID)

		http.Redirect(w, r, h.Path("account.management.user.edit", "{userID}", user.ID), http.StatusSeeOther)
	}
}

func userEditGet(h *ui.Handler) http.HandlerFunc {
	const view = "account/management/user/edit"
	h.HTML.SetViewVars(view, func(r *http.Request) (handler.Vars, error) {
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
		userID, _ := strconv.Atoi(r.PathValue("userID"))
		allowed := func(p guard.Passport) bool {
			return p.Account.CanChangeRoles(userID) || p.Account.CanSuspendUsers()
		}
		if h.RequireSignIn(w, r) || h.Forbidden(w, r, allowed) {
			return
		}

		h.HTML.View(w, r, http.StatusOK, view, nil)
	}
}

func userEditRolesGet(h *ui.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, _ := strconv.Atoi(r.PathValue("userID"))
		allowed := func(p guard.Passport) bool { return p.Account.CanChangeRoles(userID) }
		if h.RequireSignIn(w, r) || h.Forbidden(w, r, allowed) {
			return
		}

		http.Redirect(w, r, h.Path("account.management.user.edit"), http.StatusSeeOther)
	}
}

func userEditRolesPost(h *ui.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, _ := strconv.Atoi(r.PathValue("userID"))
		allowed := func(p guard.Passport) bool { return p.Account.CanChangeRoles(userID) }
		if h.RequireSignIn(w, r) || h.Forbidden(w, r, allowed) {
			return
		}

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

		var containsSuper bool
		for _, roleID := range input.RoleIDs {
			if roleID == h.SuperRole.ID {
				containsSuper = true

				if !passport.Account.CanAssignSuperRole(userID) {
					h.HTML.ErrorView(w, r, "check role ids", app.ErrForbidden, h.Session.LastView(ctx), nil)

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
			h.HTML.ErrorView(w, r, "cannot remove super role", app.ErrForbidden, h.Session.LastView(ctx), nil)

			return
		}

		_, err = h.Svc.Account.ChangeRoles(ctx, passport.Account, account.ChangeRolesInput{
			UserID:  userID,
			RoleIDs: input.RoleIDs,
			Grants:  input.Grants,
			Denials: input.Denials,
		})
		if err != nil {
			h.HTML.ErrorView(w, r, "change roles", err, h.Session.LastView(ctx), nil)

			return
		}

		h.AddFlashf(ctx, i18n.M("site.account.user_management.flash.updated", "email", user.Email))

		h.Session.SetHighlightID(ctx, user.ID)

		http.Redirect(w, r, h.PathQuery(r, "account.management.user.list"), http.StatusSeeOther)
	}
}

func userSuspendGet(h *ui.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		allowed := func(p guard.Passport) bool { return p.Account.CanSuspendUsers() }
		if h.RequireSignIn(w, r) || h.Forbidden(w, r, allowed) {
			return
		}

		http.Redirect(w, r, h.Path("account.management.user.edit"), http.StatusSeeOther)
	}
}

func userSuspendPost(h *ui.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, _ := strconv.Atoi(r.PathValue("userID"))
		allowed := func(p guard.Passport) bool { return p.Account.CanSuspendUsers() }
		if h.RequireSignIn(w, r) || h.Forbidden(w, r, allowed) {
			return
		}

		var input struct {
			SuspendedReason string `form:"suspended-reason"`
		}
		if err := httpx.DecodeRequestForm(&input, r); err != nil {
			h.HTML.ErrorView(w, r, "decode form", err, "error", nil)

			return
		}

		ctx := r.Context()
		passport := h.Passport(ctx)

		user, err := h.Repo.Account.FindUserByID(ctx, userID)
		if err != nil {
			h.HTML.ErrorView(w, r, "find user by id", err, "error", nil)

			return
		}

		if p := h.PassportByUser(ctx, user); p.IsSuper {
			h.HTML.ErrorView(w, r, "cannot suspend a user with the super role", app.ErrForbidden, h.Session.LastView(ctx), nil)

			return
		}

		wasSuspended := user.IsSuspended()

		_, err = h.Svc.Account.SuspendUser(ctx, passport.Account, userID, input.SuspendedReason)
		if err != nil {
			h.HTML.ErrorView(w, r, "suspend user", err, h.Session.LastView(ctx), nil)

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

func userUnsuspendGet(h *ui.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		allowed := func(p guard.Passport) bool { return p.Account.CanSuspendUsers() }
		if h.RequireSignIn(w, r) || h.Forbidden(w, r, allowed) {
			return
		}

		http.Redirect(w, r, h.Path("account.management.user.edit"), http.StatusSeeOther)
	}
}

func userUnsuspendPost(h *ui.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, _ := strconv.Atoi(r.PathValue("userID"))
		allowed := func(p guard.Passport) bool { return p.Account.CanSuspendUsers() }
		if h.RequireSignIn(w, r) || h.Forbidden(w, r, allowed) {
			return
		}

		ctx := r.Context()
		passport := h.Passport(ctx)

		user, err := h.Svc.Account.UnsuspendUser(ctx, passport.Account, userID)
		if err != nil {
			h.HTML.ErrorView(w, r, "unsuspend user", err, h.Session.LastView(ctx), nil)

			return
		}

		h.AddFlashf(ctx, i18n.M("site.account.user_management.flash.unsuspended", "email", user.Email))

		h.Session.SetHighlightID(ctx, user.ID)

		http.Redirect(w, r, h.PathQuery(r, "account.management.user.list"), http.StatusSeeOther)
	}
}

func userActivateGet(h *ui.Handler) http.HandlerFunc {
	const view = "account/management/user/activate"
	h.HTML.SetViewVars(view, func(r *http.Request) (handler.Vars, error) {
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
		allowed := func(p guard.Passport) bool { return p.Account.CanActivateUsers() }
		if h.RequireSignIn(w, r) || h.Forbidden(w, r, allowed) {
			return
		}

		h.HTML.View(w, r, http.StatusOK, view, nil)
	}
}

func userActivatePost(h *ui.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		allowed := func(p guard.Passport) bool { return p.Account.CanActivateUsers() }
		if h.RequireSignIn(w, r) || h.Forbidden(w, r, allowed) {
			return
		}

		ctx := r.Context()
		passport := h.Passport(ctx)

		userID, _ := strconv.Atoi(r.PathValue("userID"))
		user, err := h.Svc.Account.ActivateUser(ctx, passport.Account, userID)
		if err != nil {
			h.HTML.ErrorView(w, r, "activate user", err, h.Session.LastView(ctx), nil)

			return
		}

		h.AddFlashf(ctx, i18n.M("site.account.user_management.flash.activated", "email", user.Email))

		h.Session.SetHighlightID(ctx, user.ID)

		http.Redirect(w, r, h.PathQuery(r, "account.management.user.list"), http.StatusSeeOther)
	}
}

func userStartImpersonatePost(h *ui.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, _ := strconv.Atoi(r.PathValue("userID"))
		allowed := func(p guard.Passport) bool { return p.Account.CanImpersonateUser(userID) }
		if h.RequireSignIn(w, r) || h.Forbidden(w, r, allowed) {
			return
		}

		ctx := r.Context()

		err := auth.StartImpersonatingUser(ctx, h.Handler, userID)
		if err != nil {
			h.HTML.ErrorView(w, r, "start impersonating user", err, "error", nil)

			return
		}

		http.Redirect(w, r, h.PathQuery(r, "account.dashboard"), http.StatusSeeOther)
	}
}

func userStopImpersonatePost(h *ui.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		err := auth.StopImpersonatingUser(ctx, h.Handler)
		if err != nil {
			h.HTML.ErrorView(w, r, "stop impersonating user", err, "error", nil)

			return
		}

		http.Redirect(w, r, h.PathQuery(r, "account.dashboard"), http.StatusSeeOther)
	}
}

func userTOTPResetReviewGet(h *ui.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		allowed := func(p guard.Passport) bool { return p.Account.CanReviewTOTPResets() }
		if h.RequireSignIn(w, r) || h.Forbidden(w, r, allowed) {
			return
		}

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
		allowed := func(p guard.Passport) bool { return p.Account.CanReviewTOTPResets() }
		if h.RequireSignIn(w, r) || h.Forbidden(w, r, allowed) {
			return
		}

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
		allowed := func(p guard.Passport) bool { return p.Account.CanReviewTOTPResets() }
		if h.RequireSignIn(w, r) || h.Forbidden(w, r, allowed) {
			return
		}

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

		background.Go(func() {
			vars := handler.Vars{
				"Token":    tok,
				"ResetURL": fmt.Sprintf("%v://%v%v?token=%v", h.Scheme, h.Host, h.Path("account.totp.reset"), tok),
			}
			if err := h.SendEmail(ctx, config.SystemEmail, user.Email, "totp_reset_approved", vars); err != nil {
				logger.Error("reset TOTP: send email", "error", err)
			}
		})

		h.AddFlashf(ctx, i18n.M("site.account.user_management.flash.totp_reset_request_approved", "email", user.Email))

		http.Redirect(w, r, h.Path("account.management.user.list"), http.StatusSeeOther)
	}
}

func userTOTPResetDenyGet(h *ui.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		allowed := func(p guard.Passport) bool { return p.Account.CanReviewTOTPResets() }
		if h.RequireSignIn(w, r) || h.Forbidden(w, r, allowed) {
			return
		}

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
		allowed := func(p guard.Passport) bool { return p.Account.CanReviewTOTPResets() }
		if h.RequireSignIn(w, r) || h.Forbidden(w, r, allowed) {
			return
		}

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
