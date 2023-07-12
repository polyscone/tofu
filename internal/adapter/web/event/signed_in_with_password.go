package event

import (
	"context"

	"github.com/polyscone/tofu/internal/adapter/web/handler"
	"github.com/polyscone/tofu/internal/app/account"
	"github.com/polyscone/tofu/internal/pkg/background"
)

func SignedInWithPasswordHandler(h *handler.Handler) any {
	return func(evt account.SignedInWithPassword) {
		ctx := context.Background()
		logger := h.Logger(ctx)

		user, err := h.Repo.Account.FindUserByEmail(ctx, evt.Email)
		if err != nil {
			logger.Error("signed in with password: find user by email", "error", err)

			return
		}

		if user.HasActivatedTOTP() && user.TOTPMethod == account.TOTPMethodSMS.String() {
			background.Go(func() {
				if err := h.SendTOTPSMS(user.Email, user.TOTPTel); err != nil {
					logger.Error("signed in with password: send TOTP SMS", "error", err)
				}
			})
		}
	}
}
