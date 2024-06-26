package auth

import (
	"context"
	"fmt"
	"net/http"

	"github.com/polyscone/tofu/app"
	"github.com/polyscone/tofu/web/handler"
)

func SignUp(ctx context.Context, h *handler.Handler, w http.ResponseWriter, r *http.Request, email string) error {
	config := h.Config(ctx)

	if !config.SignUpEnabled {
		return fmt.Errorf("%w: sign up disabled", app.ErrForbidden)
	}

	_, err := h.Svc.Account.SignUp(ctx, email)

	return err
}
