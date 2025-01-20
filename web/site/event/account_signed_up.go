package event

import (
	"context"
	"fmt"
	"time"

	"github.com/polyscone/tofu/app/account"
	"github.com/polyscone/tofu/web/handler"
	"github.com/polyscone/tofu/web/site/ui"
)

func AccountSignedUpHandler(h *ui.Handler) any {
	return func(ctx context.Context, evt account.SignedUp) {
		// Sign ups through magic links and third-party services like Google/Facebook are
		// implicitly verified due to the fact they signed in with that service
		// so we don't need to verify any email addresses
		switch evt.Method {
		case account.SignUpMethodMagicLink, account.SignUpMethodGoogle, account.SignUpMethodFacebook:
			return
		}

		ctx = context.WithoutCancel(ctx)
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

		vars := handler.Vars{
			"Token":     tok,
			"VerifyURL": fmt.Sprintf("%v://%v%v?token=%v", h.Scheme, h.Host, h.Path("account.verify"), tok),
		}
		if err := h.SendEmail(ctx, config.SystemEmail, evt.Email, "verify_account", vars); err != nil {
			logger.Error("signed up: send email", "error", err)
		}
	}
}
