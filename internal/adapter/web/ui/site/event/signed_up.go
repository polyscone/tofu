package event

import (
	"context"
	"time"

	"github.com/polyscone/tofu/internal/adapter/web/handler"
	"github.com/polyscone/tofu/internal/adapter/web/ui"
	"github.com/polyscone/tofu/internal/app/account"
)

func SignedUpHandler(h *ui.Handler) any {
	return func(evt account.SignedUp) {
		// Sign ups through magic links and third-party services like Google/Facebook are
		// implicitly verified due to the fact they signed in with that service
		// so we don't need to verify any email addresses
		switch evt.Method {
		case account.SignUpMethodMagicLink, account.SignUpMethodGoogle, account.SignUpMethodFacebook:
			return
		}

		ctx := context.Background()
		logger := h.Logger(ctx)

		tok, err := h.Repo.Web.AddEmailVerificationToken(ctx, evt.Email, 2*time.Hour)
		if err != nil {
			logger.Error("signed up: add verification token", "error", err)

			return
		}

		config, err := h.Repo.System.FindConfig(ctx)
		if err != nil {
			logger.Error("signed up: find config", "error", err)

			return
		}

		vars := handler.Vars{"Token": tok}
		if err := h.SendEmail(ctx, config.SystemEmail, evt.Email, "site/verify_account", vars); err != nil {
			logger.Error("signed up: send email", "error", err)
		}
	}
}
