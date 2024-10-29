package event

import (
	"context"

	"github.com/polyscone/tofu/app/account"
	"github.com/polyscone/tofu/web/pwa/ui"
)

func SignedInHandler(h *ui.Handler) any {
	return func(ctx context.Context, evt account.SignedIn) {
		ctx = context.WithoutCancel(ctx)
		logger := h.Logger(ctx)

		user, err := h.Repo.Account.FindUserByEmail(ctx, evt.Email)
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
