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

			if err := h.SendEmail(ctx, config.SystemEmail, evt.Email, "account_activated", nil); err != nil {
				logger.Error("activated: send email", "error", err)
			}
		})
	}
}
