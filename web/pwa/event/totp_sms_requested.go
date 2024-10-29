package event

import (
	"context"

	"github.com/polyscone/tofu/web/event"
	"github.com/polyscone/tofu/web/pwa/ui"
)

func TOTPSMSRequestedHandler(h *ui.Handler) any {
	return func(ctx context.Context, evt event.TOTPSMSRequested) {
		ctx = context.WithoutCancel(ctx)
		logger := h.Logger(ctx)

		if err := h.SendTOTPSMS(evt.Email, evt.Tel); err != nil {
			logger.Error("TOTP SMS requested: send TOTP SMS", "error", err)
		}
	}
}
