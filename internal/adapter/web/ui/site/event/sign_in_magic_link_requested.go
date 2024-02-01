package event

import (
	"context"

	"github.com/polyscone/tofu/internal/adapter/web/event"
	"github.com/polyscone/tofu/internal/adapter/web/handler"
	"github.com/polyscone/tofu/internal/adapter/web/ui"
	"github.com/polyscone/tofu/internal/pkg/human"
)

func SignInMagicLinkRequestedHandler(h *ui.Handler) any {
	return func(evt event.SignInMagicLinkRequested) {
		ctx := context.Background()
		logger := h.Logger(ctx)

		config, err := h.Repo.System.FindConfig(ctx)
		if err != nil {
			logger.Error("sign in magic link: find config", "error", err)

			return
		}

		tok, err := h.Repo.Web.AddSignInMagicLinkToken(ctx, evt.Email, evt.TTL)
		if err != nil {
			logger.Error("sign in magic link: add sign in magic link token", "error", err)

			return
		}

		vars := handler.Vars{
			"Token": tok,
			"TTL":   human.Duration(evt.TTL),
		}
		if err := h.SendEmail(ctx, config.SystemEmail, evt.Email, "site/sign_in_magic_link", vars); err != nil {
			logger.Error("sign in magic link: send email", "error", err)
		}
	}
}
