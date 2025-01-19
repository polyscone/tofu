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

func Verify(ctx context.Context, h *handler.Handler, w http.ResponseWriter, r *http.Request, token, password, passwordCheck string) (string, account.VerifyUserBehavior, error) {
	config := h.Config(ctx)

	email, err := h.Repo.Web.FindEmailVerificationTokenEmail(ctx, token)
	if err != nil {
		if errors.Is(err, app.ErrNotFound) {
			err = fmt.Errorf("%w: %w", app.ErrInvalidInput, err)
		}

		return "", 0, fmt.Errorf("find verification token email: %w", err)
	}

	behavior := account.VerifyUserActivate
	if !config.SignUpAutoActivateEnabled {
		behavior = account.VerifyUserOnly
	}

	_, err = h.Svc.Account.VerifyUser(ctx, account.VerifyUserInput{
		Email:         email,
		Password:      password,
		PasswordCheck: passwordCheck,
		Behavior:      behavior,
	})
	if err != nil {
		return "", 0, fmt.Errorf("verify user: %w", err)
	}

	err = h.Repo.Web.ConsumeEmailVerificationToken(ctx, token)
	if err != nil {
		return "", 0, fmt.Errorf("consume verification token: %w", err)
	}

	return email, behavior, nil
}
