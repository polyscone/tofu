package ui

import (
	"context"
	"time"

	"github.com/polyscone/tofu/internal/adapter/web/ui/handler"
	"github.com/polyscone/tofu/internal/app/account"
	"github.com/polyscone/tofu/internal/pkg/background"
	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/logger"
)

func accountSignedInWithPasswordHandler(tenant *handler.Tenant, h *handler.Handler) any {
	return func(evt account.SignedInWithPassword) {
		ctx := context.Background()

		user, err := h.Store.Account.FindUserByEmail(ctx, evt.Email)
		if err != nil {
			logger.PrintError(errors.Tracef(err))

			return
		}

		if user.HasActivatedTOTP() && user.TOTPMethod == account.TOTPMethodSMS.String() {
			background.Go(func() {
				if err := h.SendTOTPSMS(user.Email, user.TOTPTel); err != nil {
					logger.PrintError(errors.Tracef(err))
				}
			})
		}
	}
}

func accountDisabledTOTPHandler(tenant *handler.Tenant, h *handler.Handler) any {
	return func(evt account.DisabledTOTP) {
		background.Go(func() {
			ctx := context.Background()

			recipients := handler.EmailRecipients{
				From: tenant.Email.From,
				To:   []string{evt.Email},
			}
			if err := h.SendEmail(ctx, recipients, "disabled_totp", nil); err != nil {
				logger.PrintError(errors.Tracef(err))
			}
		})
	}
}

func accountSignedUpHandler(tenant *handler.Tenant, h *handler.Handler) any {
	return func(evt account.SignedUp) {
		background.Go(func() {
			ctx := context.Background()

			tok, err := tenant.Store.Web.AddActivationToken(ctx, evt.Email, 48*time.Hour)
			if err != nil {
				logger.PrintError(errors.Tracef(err))

				return
			}

			recipients := handler.EmailRecipients{
				From: tenant.Email.From,
				To:   []string{evt.Email},
			}
			vars := handler.Vars{
				"Token": tok,
			}
			if err := h.SendEmail(ctx, recipients, "activate_account", vars); err != nil {
				logger.PrintError(errors.Tracef(err))
			}
		})
	}
}
