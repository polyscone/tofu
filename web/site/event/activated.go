package event

import (
	"context"
	"fmt"

	"github.com/polyscone/tofu/app/account"
	"github.com/polyscone/tofu/web/handler"
	"github.com/polyscone/tofu/web/site/ui"
)

func ActivatedHandler(h *ui.Handler) any {
	return func(ctx context.Context, evt account.Activated) {
		ctx = context.WithoutCancel(ctx)
		logger := h.Logger(ctx)

		config, err := h.Repo.System.FindConfig(ctx)
		if err != nil {
			logger.Error("activated: find config", "error", err)

			return
		}

		vars := handler.Vars{
			"HasPassword": evt.HasPassword,
			"SignInURL":   fmt.Sprintf("%v://%v%v", h.Scheme, h.Host, h.Path("account.sign_in")),
		}
		if err := h.SendEmail(ctx, config.SystemEmail, evt.Email, "account_activated", vars); err != nil {
			logger.Error("activated: send email", "error", err)
		}
	}
}
