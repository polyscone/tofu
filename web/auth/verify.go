package auth

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/polyscone/tofu/app"
	"github.com/polyscone/tofu/app/account"
	"github.com/polyscone/tofu/web/handler"
)

func Verify(ctx context.Context, h *handler.Handler, w http.ResponseWriter, r *http.Request, token, password, passwordCheck string) (string, account.VerifyUserBehaviour, error) {
	config := h.Config(ctx)

	email, err := h.Repo.Web.FindEmailVerificationTokenEmail(ctx, token)
	if err != nil {
		if errors.Is(err, app.ErrNotFound) {
			err = fmt.Errorf("%w: %w", app.ErrInvalidInput, err)
		}

		return "", 0, fmt.Errorf("find verification token email: %w", err)
	}

	behaviour := account.VerifyUserActivate
	if !config.SignUpAutoActivateEnabled {
		behaviour = account.VerifyUserOnly
	}

	err = h.Svc.Account.VerifyUser(ctx, email, password, passwordCheck, behaviour)
	if err != nil {
		return "", 0, fmt.Errorf("verify user: %w", err)
	}

	err = h.Repo.Web.ConsumeEmailVerificationToken(ctx, token)
	if err != nil {
		return "", 0, fmt.Errorf("consume verification token: %w", err)
	}

	return email, behaviour, nil
}
