package event

import (
	"context"

	"github.com/polyscone/tofu/internal/adapter/web/handler"
	"github.com/polyscone/tofu/internal/adapter/web/ui"
	"github.com/polyscone/tofu/internal/app/account"
)

func ActivatedHandler(h *ui.Handler) any {
	return func(evt account.Activated) {
		ctx := context.Background()
		logger := h.Logger(ctx)

		config, err := h.Repo.System.FindConfig(ctx)
		if err != nil {
			logger.Error("activated: find config", "error", err)

			return
		}

		emailTemplate := "site/account_activated"
		if evt.System == "pwa" {
			emailTemplate = "pwa/account_activated"
		}

		vars := handler.Vars{"HasPassword": evt.HasPassword}
		if err := h.SendEmail(ctx, config.SystemEmail, evt.Email, emailTemplate, vars); err != nil {
			logger.Error("activated: send email", "error", err)
		}
	}
}

type changeRoleGuard struct{}

func (g changeRoleGuard) CanChangeRoles(userID int) bool     { return true }
func (g changeRoleGuard) CanAssignSuperRole(userID int) bool { return true }

func ImmediateActivatedHandler(h *ui.Handler) any {
	return func(evt account.Activated) {
		ctx := context.Background()
		logger := h.Logger(ctx)

		superUserCount, err := h.Repo.Account.CountUsersByRoleID(ctx, account.SuperRole.ID)
		if err != nil {
			logger.Error("immediate activated: count users by role id", "error", err)

			return
		}
		if superUserCount == 0 {
			user, err := h.Repo.Account.FindUserByEmail(ctx, evt.Email)
			if err != nil {
				logger.Error("immediate activated: find user by email", "error", err)

				return
			}

			err = h.Svc.Account.ChangeRoles(ctx, changeRoleGuard{}, user.ID, []int{account.SuperRole.ID}, nil, nil)
			if err != nil {
				logger.Error("immediate activated: change roles", "error", err)

				return
			}
		}
	}
}
