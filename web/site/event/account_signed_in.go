package event

import (
	"context"
	"time"

	"github.com/polyscone/tofu/app/account"
	"github.com/polyscone/tofu/web/site/ui"
)

func AccountSignedInHandler(h *ui.Handler) any {
	return func(ctx context.Context, data account.SignedIn, createdAt time.Time) {
		ctx = context.WithoutCancel(ctx)
		logger := h.Logger(ctx)

		user, err := h.Repo.Account.FindUserByEmail(ctx, data.Email)
		if err != nil {
			logger.Error("signed in: find user by email", "error", err)

			return
		}

		if user.HasActivatedTOTP() && user.TOTPMethod == account.TOTPMethodSMS.String() {
			if err := h.SendTOTPSMS(user.Email, user.TOTPTel); err != nil {
				logger.Error("signed in: send TOTP SMS", "error", err)
			}
		}
	}
}
