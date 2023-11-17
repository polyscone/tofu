package event

import (
	"context"
	"time"

	"github.com/polyscone/tofu/internal/adapter/web/handler"
	"github.com/polyscone/tofu/internal/adapter/web/ui"
	"github.com/polyscone/tofu/internal/app/account"
	"github.com/polyscone/tofu/internal/pkg/background"
)

func SignedUpHandler(h *ui.Handler) any {
	return func(evt account.SignedUp) {
		background.Go(func() {
			ctx := context.Background()
			logger := h.Logger(ctx)

			tok, err := h.Repo.Web.AddEmailVerificationToken(ctx, evt.Email, 2*time.Hour)
			if err != nil {
				logger.Error("signed up: add verification token", "error", err)

				return
			}

			config, err := h.Repo.System.FindConfig(ctx)
			if err != nil {
				logger.Error("signed up: find config", "error", err)

				return
			}

			vars := handler.Vars{"Token": tok}
			if err := h.SendEmail(ctx, config.SystemEmail, evt.Email, "site/verify_account", vars); err != nil {
				logger.Error("signed up: send email", "error", err)
			}
		})
	}
}
