package event

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/polyscone/tofu/app"
	"github.com/polyscone/tofu/internal/background"
	"github.com/polyscone/tofu/web/event"
	"github.com/polyscone/tofu/web/handler"
	"github.com/polyscone/tofu/web/site/ui"
)

func WebPasswordResetRequestedHandler(h *ui.Handler) any {
	return func(ctx context.Context, data event.PasswordResetRequested, createdAt time.Time) {
		ctx = context.WithoutCancel(ctx)
		logger := h.Logger(ctx)

		config, err := h.Repo.System.FindConfig(ctx)
		if err != nil {
			logger.Error("reset password: find config", "error", err)

			return
		}

		_, err = h.Repo.Account.FindUserByEmail(ctx, data.Email)
		switch {
		case err == nil:
			tok, err := h.Repo.Web.AddResetPasswordToken(ctx, data.Email, 2*time.Hour)
			if err != nil {
				logger.Error("reset password: add reset password token", "error", err)

				return
			}

			background.Go(func() {
				vars := handler.Vars{
					"Token":    tok,
					"ResetURL": fmt.Sprintf("%v://%v%v?token=%v", h.Scheme, h.Host, h.Path("account.reset_password.new_password"), tok),
				}
				if err := h.SendEmail(ctx, config.SystemEmail, data.Email, "reset_password", vars); err != nil {
					logger.Error("reset password: send email", "error", err)
				}
			})

		case errors.Is(err, app.ErrNotFound):
			if config.SignUpEnabled {
				background.Go(func() {
					vars := handler.Vars{
						"SignUpURL": fmt.Sprintf("%v://%v%v", h.Scheme, h.Host, h.Path("account.sign_up")),
					}
					if err := h.SendEmail(ctx, config.SystemEmail, data.Email, "reset_password_sign_up", vars); err != nil {
						logger.Error("reset password: send email", "error", err)
					}
				})
			}

		default:
			logger.Error("reset password: find user by email", "error", err)
		}
	}
}
