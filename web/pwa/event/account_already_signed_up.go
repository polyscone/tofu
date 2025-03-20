package event

import (
	"context"
	"time"

	"github.com/polyscone/tofu/app/account"
	"github.com/polyscone/tofu/internal/background"
	"github.com/polyscone/tofu/web/handler"
	"github.com/polyscone/tofu/web/pwa/ui"
)

func AccountAlreadySignedUpHandler(h *ui.Handler) any {
	return func(ctx context.Context, data account.AlreadySignedUp, createdAt time.Time) {
		ctx = context.WithoutCancel(ctx)
		logger := h.Logger(ctx)

		tok, err := h.Repo.Web.AddResetPasswordToken(ctx, data.Email, 2*time.Hour)
		if err != nil {
			logger.Error("already signed up: add reset password token", "error", err)

			return
		}

		config, err := h.Repo.System.FindConfig(ctx)
		if err != nil {
			logger.Error("already signed up: find config", "error", err)

			return
		}

		background.Go(func() {
			vars := handler.Vars{
				"Token":       tok,
				"HasPassword": data.HasPassword,
			}
			if err := h.SendEmail(ctx, config.SystemEmail, data.Email, "sign_up_reset_password", vars); err != nil {
				logger.Error("already signed up: send email", "error", err)
			}
		})
	}
}
