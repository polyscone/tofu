package event

import (
	"context"
	"time"

	"github.com/polyscone/tofu/internal/background"
	"github.com/polyscone/tofu/web/event"
	"github.com/polyscone/tofu/web/handler"
	"github.com/polyscone/tofu/web/pwa/ui"
)

func WebSignInMagicLinkRequestedHandler(h *ui.Handler) any {
	return func(ctx context.Context, data event.SignInMagicLinkRequested, createdAt time.Time) {
		ctx = context.WithoutCancel(ctx)
		logger := h.Logger(ctx)

		config, err := h.Repo.System.FindConfig(ctx)
		if err != nil {
			logger.Error("sign in magic link: find config", "error", err)

			return
		}

		tok, err := h.Repo.Web.AddSignInMagicLinkToken(ctx, data.Email, data.TTL)
		if err != nil {
			logger.Error("sign in magic link: add sign in magic link token", "error", err)

			return
		}

		background.Go(func() {
			vars := handler.Vars{
				"Token": tok,
				"TTL":   data.TTL,
			}
			if err := h.SendEmail(ctx, config.SystemEmail, data.Email, "sign_in_magic_link", vars); err != nil {
				logger.Error("sign in magic link: send email", "error", err)
			}
		})
	}
}
