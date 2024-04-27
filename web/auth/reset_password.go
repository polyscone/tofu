package auth

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/polyscone/tofu/app"
	"github.com/polyscone/tofu/web/handler"
)

func ResetPassword(ctx context.Context, h *handler.Handler, w http.ResponseWriter, r *http.Request, token, newPassword, newPasswordCheck string) (string, error) {
	email, err := h.Repo.Web.FindResetPasswordTokenEmail(ctx, token)
	if err != nil {
		if errors.Is(err, app.ErrNotFound) {
			err = fmt.Errorf("%w: %w", app.ErrInvalidInput, err)
		}

		return "", fmt.Errorf("find reset password token email: %w", err)
	}

	user, err := h.Repo.Account.FindUserByEmail(ctx, email)
	if err != nil {
		return "", fmt.Errorf("find user by email: %w", err)
	}

	passport, err := h.PassportByEmail(ctx, email)
	if err != nil {
		return "", fmt.Errorf("passport by email: %w", err)
	}

	err = h.Svc.Account.ResetPassword(ctx, passport.Account, user.ID, newPassword, newPasswordCheck)
	if err != nil {
		return "", err
	}

	err = h.Repo.Web.ConsumeResetPasswordToken(ctx, token)
	if err != nil {
		return "", fmt.Errorf("consume reset password token: %w", err)
	}

	return email, nil
}
