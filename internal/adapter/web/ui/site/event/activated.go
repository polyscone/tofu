package event

import (
	"context"

	"github.com/polyscone/tofu/internal/adapter/web/ui"
	"github.com/polyscone/tofu/internal/app/account"
	"github.com/polyscone/tofu/internal/pkg/background"
)

func ActivatedHandler(h *ui.Handler) any {
	return func(evt account.Activated) {
		background.Go(func() {
			ctx := context.Background()
			logger := h.Logger(ctx)

			config, err := h.Repo.System.FindConfig(ctx)
			if err != nil {
				logger.Error("activated: find config", "error", err)

				return
			}

			user, err := h.Repo.Account.FindUserByEmail(ctx, evt.Email)
			if err != nil {
				logger.Error("activated: find user by email", "error", err)

				return
			}

			emailTemplate := "site/account_activated"
			if user.SignedUpSystem == "pwa" {
				emailTemplate = "pwa/account_activated"
			}

			if err := h.SendEmail(ctx, config.SystemEmail, evt.Email, emailTemplate, nil); err != nil {
				logger.Error("activated: send email", "error", err)
			}
		})
	}
}
