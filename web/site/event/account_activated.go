package event

import (
	"context"
	"fmt"
	"time"

	"github.com/polyscone/tofu/app/account"
	"github.com/polyscone/tofu/web/handler"
	"github.com/polyscone/tofu/web/site/ui"
)

func AccountActivatedHandler(h *ui.Handler) any {
	return func(ctx context.Context, data account.Activated, createdAt time.Time) {
		ctx = context.WithoutCancel(ctx)
		logger := h.Logger(ctx)

		config, err := h.Repo.System.FindConfig(ctx)
		if err != nil {
			logger.Error("activated: find config", "error", err)

			return
		}

		vars := handler.Vars{
			"HasPassword": data.HasPassword,
			"SignInURL":   fmt.Sprintf("%v://%v%v", h.Scheme, h.Host, h.Path("account.sign_in")),
		}
		if err := h.SendEmail(ctx, config.SystemEmail, data.Email, "account_activated", vars); err != nil {
			logger.Error("activated: send email", "error", err)
		}
	}
}
