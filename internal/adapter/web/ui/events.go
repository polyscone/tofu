package ui

import (
	"context"
	"time"

	"github.com/polyscone/tofu/internal/adapter/web/event"
	"github.com/polyscone/tofu/internal/adapter/web/handler"
	"github.com/polyscone/tofu/internal/pkg/background"
	"github.com/polyscone/tofu/internal/pkg/logger"
	"github.com/polyscone/tofu/internal/pkg/valobj/text"
	"github.com/polyscone/tofu/internal/port/account"
)

func accountRegisteredHandler(tenant *handler.Tenant, svc *handler.Services) any {
	return func(evt account.Registered) {
		background.Go(func() {
			ctx := context.Background()

			email, err := text.NewEmail(evt.Email)
			if err != nil {
				logger.PrintError(err)

				return
			}

			tok, err := tenant.Tokens.AddActivationToken(ctx, email, 48*time.Hour)
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
	return func(evt event.ResetPasswordRequested) {
		background.Go(func() {
			ctx := context.Background()

			email, err := text.NewEmail(evt.Email)
			if err != nil {
				logger.PrintError(err)

				return
			}

			tok, err := tenant.Tokens.AddResetPasswordToken(ctx, email, 2*time.Hour)
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
