package event

import (
	"context"

	"github.com/polyscone/tofu/app/account"
	"github.com/polyscone/tofu/web/ui"
)

func TOTPDisabledHandler(h *ui.Handler) any {
	return func(evt account.TOTPDisabled) {
		ctx := context.Background()
		logger := h.Logger(ctx)

		config, err := h.Repo.System.FindConfig(ctx)
		if err != nil {
			logger.Error("disabled TOTP: find config", "error", err)

			return
		}

		if err := h.SendEmail(ctx, config.SystemEmail, evt.Email, "site/totp_disabled", nil); err != nil {
			logger.Error("disabled TOTP: send email", "error", err)
		}
	}
}
