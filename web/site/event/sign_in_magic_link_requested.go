package event

import (
	"context"
	"fmt"

	"github.com/polyscone/tofu/web/event"
	"github.com/polyscone/tofu/web/handler"
	"github.com/polyscone/tofu/web/site/ui"
)

func SignInMagicLinkRequestedHandler(h *ui.Handler) any {
	return func(ctx context.Context, evt event.SignInMagicLinkRequested) {
		ctx = context.WithoutCancel(ctx)
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
			"Token":     tok,
			"TTL":       evt.TTL,
			"SignInURL": fmt.Sprintf("%v://%v%v?token=%v", h.Scheme, h.Host, h.Path("account.sign_in.magic_link"), tok),
		}
		if err := h.SendEmail(ctx, config.SystemEmail, evt.Email, "sign_in_magic_link", vars); err != nil {
			logger.Error("sign in magic link: send email", "error", err)
		}
	}
}
