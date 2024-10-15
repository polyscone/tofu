package event

import (
	"context"

	"github.com/polyscone/tofu/app/account"
	"github.com/polyscone/tofu/web/handler"
	"github.com/polyscone/tofu/web/site/ui"
)

func ActivatedHandler(h *ui.Handler) any {
	return func(evt account.Activated) {
		ctx := context.Background()
		logger := h.Logger(ctx)

		config, err := h.Repo.System.FindConfig(ctx)
		if err != nil {
			logger.Error("activated: find config", "error", err)

			return
		}

		emailTemplate := "account_activated"
		if evt.System == "pwa" {
			emailTemplate = "account_activated"
		}

		vars := handler.Vars{"HasPassword": evt.HasPassword}
		if err := h.SendEmail(ctx, config.SystemEmail, evt.Email, emailTemplate, vars); err != nil {
			logger.Error("activated: send email", "error", err)
		}
	}
}
