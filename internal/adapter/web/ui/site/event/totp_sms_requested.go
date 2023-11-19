package event

import (
	"context"

	"github.com/polyscone/tofu/internal/adapter/web/event"
	"github.com/polyscone/tofu/internal/adapter/web/ui"
	"github.com/polyscone/tofu/internal/pkg/background"
)

func TOTPSMSRequestedHandler(h *ui.Handler) any {
	return func(evt event.TOTPSMSRequested) {
		background.Go(func() {
			ctx := context.Background()
			logger := h.Logger(ctx)

			if err := h.SendTOTPSMS(evt.Email, evt.Tel); err != nil {
				logger.Error("TOTP SMS requested: send TOTP SMS", "error", err)
			}
		})
	}
}
