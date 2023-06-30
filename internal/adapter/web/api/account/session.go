package account

import (
	"context"
	"net/http"

	"github.com/polyscone/tofu/internal/adapter/web/api"
	"github.com/polyscone/tofu/internal/adapter/web/sess"
	"github.com/polyscone/tofu/internal/pkg/http/router"
)

func Session(h *api.Handler, mux *router.ServeMux) {
	mux.Get("/session", sessionGet(h))
}

func sessionGet(h *api.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		h.JSON(w, r, SessionData(ctx, h))
	}
}

func SessionData(ctx context.Context, h *api.Handler) map[string]any {
	return map[string]any{
		"isSignedIn":     h.Sessions.GetBool(ctx, sess.IsSignedIn),
		"isAwaitingTOTP": h.Sessions.GetBool(ctx, sess.IsAwaitingTOTP),
		"totpMethod":     h.Sessions.GetString(ctx, sess.TOTPMethod),
	}
}
