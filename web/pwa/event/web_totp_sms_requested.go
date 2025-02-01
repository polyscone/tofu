package event

import (
	"context"
	"time"

	"github.com/polyscone/tofu/web/event"
	"github.com/polyscone/tofu/web/pwa/ui"
)

func WebTOTPSMSRequestedHandler(h *ui.Handler) any {
	return func(ctx context.Context, data event.TOTPSMSRequested, createdAt time.Time) {
		ctx = context.WithoutCancel(ctx)
		logger := h.Logger(ctx)

		if err := h.SendTOTPSMS(data.Email, data.Tel); err != nil {
			logger.Error("TOTP SMS requested: send TOTP SMS", "error", err)
		}
	}
}
