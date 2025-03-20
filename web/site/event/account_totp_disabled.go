package event

import (
	"context"
	"time"

	"github.com/polyscone/tofu/app/account"
	"github.com/polyscone/tofu/internal/background"
	"github.com/polyscone/tofu/web/site/ui"
)

func AccountTOTPDisabledHandler(h *ui.Handler) any {
	return func(ctx context.Context, data account.TOTPDisabled, createdAt time.Time) {
		ctx = context.WithoutCancel(ctx)
		logger := h.Logger(ctx)

		config, err := h.Repo.System.FindConfig(ctx)
		if err != nil {
			logger.Error("disabled TOTP: find config", "error", err)

			return
		}

		background.Go(func() {
			if err := h.SendEmail(ctx, config.SystemEmail, data.Email, "totp_disabled", nil); err != nil {
				logger.Error("disabled TOTP: send email", "error", err)
			}
		})
	}
}
