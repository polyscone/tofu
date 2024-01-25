package event

import (
	"context"
	"time"

	"github.com/polyscone/tofu/internal/adapter/web/handler"
	"github.com/polyscone/tofu/internal/adapter/web/ui"
	"github.com/polyscone/tofu/internal/app/account"
	"github.com/polyscone/tofu/internal/pkg/background"
)

func AlreadySignedUpHandler(h *ui.Handler) any {
	return func(evt account.AlreadySignedUp) {
		background.Go(func() {
			ctx := context.Background()
			logger := h.Logger(ctx)

			tok, err := h.Repo.Web.AddResetPasswordToken(ctx, evt.Email, 2*time.Hour)
			if err != nil {
				logger.Error("already signed up: add reset password token", "error", err)

				return
			}

			config, err := h.Repo.System.FindConfig(ctx)
			if err != nil {
				logger.Error("already signed up: find config", "error", err)

				return
			}

			vars := handler.Vars{
				"Token":       tok,
				"HasPassword": evt.HasPassword,
			}
			if err := h.SendEmail(ctx, config.SystemEmail, evt.Email, "pwa/sign_up_reset_password", vars); err != nil {
				logger.Error("already signed up: send email", "error", err)
			}
		})
	}
}
