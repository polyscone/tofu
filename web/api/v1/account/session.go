package account

import (
	"context"
	"net/http"

	"github.com/polyscone/tofu/httpx/router"
	"github.com/polyscone/tofu/web/api"
	"github.com/polyscone/tofu/web/sess"
)

func RegisterSessionHandlers(h *api.Handler, mux *router.ServeMux) {
	mux.HandleFunc("GET /account/session", sessionGet(h))
}

func sessionGet(h *api.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		h.JSON(w, r, http.StatusOK, SessionData(ctx, h))
	}
}

func SessionData(ctx context.Context, h *api.Handler) map[string]any {
	config := h.Config(ctx)
	user := h.User(ctx)

	isSignedIn := h.Sessions.GetBool(ctx, sess.IsSignedIn)

	return map[string]any{
		"isSignedIn":     isSignedIn,
		"isAwaitingTOTP": h.Sessions.GetBool(ctx, sess.IsAwaitingTOTP),
		"totpMethod":     h.Sessions.GetString(ctx, sess.TOTPMethod),
		"isTOTPRequired": isSignedIn && config.TOTPRequired && !user.HasActivatedTOTP(),
	}
}
