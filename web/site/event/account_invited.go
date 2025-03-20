package event

import (
	"context"
	"fmt"
	"time"

	"github.com/polyscone/tofu/app/account"
	"github.com/polyscone/tofu/internal/background"
	"github.com/polyscone/tofu/web/handler"
	"github.com/polyscone/tofu/web/site/ui"
)

func AccountInvitedHandler(h *ui.Handler) any {
	return func(ctx context.Context, data account.Invited, createdAt time.Time) {
		ctx = context.WithoutCancel(ctx)
		logger := h.Logger(ctx)

		tok, err := h.Repo.Web.AddEmailVerificationToken(ctx, data.Email, 48*time.Hour)
		if err != nil {
			logger.Error("invited: add verification token", "error", err)

			return
		}

		config, err := h.Repo.System.FindConfig(ctx)
		if err != nil {
			logger.Error("invited: find config", "error", err)

			return
		}

		background.Go(func() {
			vars := handler.Vars{
				"Token":     tok,
				"VerifyURL": fmt.Sprintf("%v://%v%v?token=%v", h.Scheme, h.Host, h.Path("account.verify"), tok),
			}
			if err := h.SendEmail(ctx, config.SystemEmail, data.Email, "invite_verify_account", vars); err != nil {
				logger.Error("invited: send email", "error", err)
			}
		})
	}
}
