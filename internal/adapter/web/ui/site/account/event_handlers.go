package account

import (
	"context"
	"time"

	"github.com/polyscone/tofu/internal/adapter/web/handler"
	"github.com/polyscone/tofu/internal/adapter/web/ui"
	"github.com/polyscone/tofu/internal/app/account"
	"github.com/polyscone/tofu/internal/pkg/background"
)

func SignedInWithPasswordHandler(h *ui.Handler) any {
	return func(evt account.SignedInWithPassword) {
		ctx := context.Background()
		logger := h.Logger(ctx)

		user, err := h.Repo.Account.FindUserByEmail(ctx, evt.Email)
		if err != nil {
			logger.Error("signed in with password: find user by email", "error", err)

			return
		}

		if user.HasActivatedTOTP() && user.TOTPMethod == account.TOTPMethodSMS.String() {
			background.Go(func() {
				if err := h.SendTOTPSMS(user.Email, user.TOTPTel); err != nil {
					logger.Error("signed in with password: send TOTP SMS", "error", err)
				}
			})
		}
	}
}

func SignedUpHandler(h *ui.Handler) any {
	return func(evt account.SignedUp) {
		background.Go(func() {
			ctx := context.Background()
			logger := h.Logger(ctx)

			tok, err := h.Repo.Web.AddActivationToken(ctx, evt.Email, 2*time.Hour)
			if err != nil {
				logger.Error("signed up: add activation token", "error", err)

				return
			}

			config, err := h.Repo.System.FindConfig(ctx)
			if err != nil {
				logger.Error("signed up: find config", "error", err)

				return
			}

			vars := handler.Vars{"Token": tok}
			if err := h.SendEmail(ctx, config.SystemEmail, evt.Email, "activate_account", vars); err != nil {
				logger.Error("signed up: send email", "error", err)
			}
		})
	}
}

func TOTPDisabledHandler(h *ui.Handler) any {
	return func(evt account.TOTPDisabled) {
		background.Go(func() {
			ctx := context.Background()
			logger := h.Logger(ctx)

			config, err := h.Repo.System.FindConfig(ctx)
			if err != nil {
				logger.Error("disabled TOTP: find config", "error", err)

				return
			}

			if err := h.SendEmail(ctx, config.SystemEmail, evt.Email, "totp_disabled", nil); err != nil {
				logger.Error("disabled TOTP: send email", "error", err)
			}
		})
	}
}
