package ui

import (
	"context"
	"time"

	"github.com/polyscone/tofu/internal/adapter/web/ui/handler"
	"github.com/polyscone/tofu/internal/app/account"
	"github.com/polyscone/tofu/internal/pkg/background"
	"golang.org/x/exp/slog"
)

func accountSignedInWithPasswordHandler(tenant *handler.Tenant, h *handler.Handler) any {
	return func(evt account.SignedInWithPassword) {
		ctx := context.Background()

		user, err := h.Repo.Account.FindUserByEmail(ctx, evt.Email)
		if err != nil {
			slog.Error("signed in with password: find user by email", "error", err)

			return
		}

		if user.HasActivatedTOTP() && user.TOTPMethod == account.TOTPMethodSMS.String() {
			background.Go(func() {
				if err := h.SendTOTPSMS(user.Email, user.TOTPTel); err != nil {
					slog.Error("signed in with password: send TOTP SMS", "error", err)
				}
			})
		}
	}
}

func accountDisabledTOTPHandler(tenant *handler.Tenant, h *handler.Handler) any {
	return func(evt account.DisabledTOTP) {
		background.Go(func() {
			ctx := context.Background()

			config, err := h.Repo.System.FindConfig(ctx)
			if err != nil {
				slog.Error("disabled TOTP: find config", "error", err)

				return
			}

			recipients := handler.EmailRecipients{
				From: config.SystemEmail,
				To:   []string{evt.Email},
			}
			if err := h.SendEmail(ctx, recipients, "disabled_totp", nil); err != nil {
				slog.Error("disabled TOTP: send email", "error", err)
			}
		})
	}
}

func accountSignedUpHandler(tenant *handler.Tenant, h *handler.Handler) any {
	return func(evt account.SignedUp) {
		background.Go(func() {
			ctx := context.Background()

			tok, err := tenant.Repo.Web.AddActivationToken(ctx, evt.Email, 2*time.Hour)
			if err != nil {
				slog.Error("signed up: add activation token", "error", err)

				return
			}

			config, err := h.Repo.System.FindConfig(ctx)
			if err != nil {
				slog.Error("signed up: find config", "error", err)

				return
			}

			recipients := handler.EmailRecipients{
				From: config.SystemEmail,
				To:   []string{evt.Email},
			}
			vars := handler.Vars{
				"Token": tok,
			}
			if err := h.SendEmail(ctx, recipients, "activate_account", vars); err != nil {
				slog.Error("signed up: send email", "error", err)
			}
		})
	}
}
