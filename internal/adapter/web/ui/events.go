package ui

import (
	"context"
	"time"

	"github.com/polyscone/tofu/internal/adapter/web/handler"
	"github.com/polyscone/tofu/internal/app/account"
	"github.com/polyscone/tofu/internal/pkg/background"
	"github.com/polyscone/tofu/internal/pkg/logger"
)

func accountRegisteredHandler(tenant *handler.Tenant, svc *handler.Services) any {
	return func(evt account.Registered) {
		background.Go(func() {
			ctx := context.Background()

			tok, err := tenant.Repo.Web.AddActivationToken(ctx, evt.Email, 48*time.Hour)
			if err != nil {
				logger.PrintError(err)

				return
			}

			recipients := handler.EmailRecipients{
				From: "noreply@example.com",
				To:   []string{evt.Email},
			}
			vars := handler.Vars{
				"Token": tok,
			}
			if err := svc.SendEmail(ctx, recipients, "activate_account", vars); err != nil {
				logger.PrintError(err)
			}
		})
	}
}

func accountResetPasswordRequestedHandler(tenant *handler.Tenant, svc *handler.Services) any {
	return func(evt handler.ResetPasswordRequested) {
		background.Go(func() {
			ctx := context.Background()

			tok, err := tenant.Repo.Web.AddResetPasswordToken(ctx, evt.Email, 2*time.Hour)
			if err != nil {
				logger.PrintError(err)

				return
			}

			recipients := handler.EmailRecipients{
				From: "noreply@example.com",
				To:   []string{evt.Email},
			}
			vars := handler.Vars{
				"Token": tok,
			}
			if err := svc.SendEmail(ctx, recipients, "reset_password", vars); err != nil {
				logger.PrintError(err)
			}
		})
	}
}

func accountAuthenticateWithPasswordHandler(tenant *handler.Tenant, svc *handler.Services) any {
	return func(evt account.AuthenticatedWithPassword) {
		ctx := context.Background()

		user, err := svc.Repo.Account.FindUserByEmail(ctx, evt.Email)
		if err != nil {
			logger.PrintError(err)

			return
		}

		if user.HasVerifiedTOTP() && user.TOTPMethod == account.TOTPMethodSMS.String() {
			background.Go(func() {
				if err := svc.SendTOTPSMS(evt.Email); err != nil {
					logger.PrintError(err)
				}
			})
		}
	}
}

func accountTOTPSMSRequestedHandler(tenant *handler.Tenant, svc *handler.Services) any {
	return func(evt handler.TOTPSMSRequested) {
		background.Go(func() {
			if err := svc.SendTOTPSMS(evt.Email); err != nil {
				logger.PrintError(err)
			}
		})
	}
}
