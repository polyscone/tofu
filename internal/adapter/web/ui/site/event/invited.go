package event

import (
	"context"
	"time"

	"github.com/polyscone/tofu/internal/adapter/web/handler"
	"github.com/polyscone/tofu/internal/adapter/web/ui"
	"github.com/polyscone/tofu/internal/app/account"
	"github.com/polyscone/tofu/internal/pkg/background"
)

func InvitedHandler(h *ui.Handler) any {
	return func(evt account.Invited) {
		background.Go(func() {
			ctx := context.Background()
			logger := h.Logger(ctx)

			tok, err := h.Repo.Web.AddActivationToken(ctx, evt.Email, 48*time.Hour)
			if err != nil {
				logger.Error("invited: add activation token", "error", err)

				return
			}

			config, err := h.Repo.System.FindConfig(ctx)
			if err != nil {
				logger.Error("invited: find config", "error", err)

				return
			}

			vars := handler.Vars{"Token": tok}
			if err := h.SendEmail(ctx, config.SystemEmail, evt.Email, "invite_activate_account", vars); err != nil {
				logger.Error("invited: send email", "error", err)
			}
		})
	}
}
