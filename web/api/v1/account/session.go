package account

import (
	"context"
	"net/http"

	"github.com/polyscone/tofu/httpx/router"
	"github.com/polyscone/tofu/web/api"
)

func RegisterSessionHandlers(h *api.Handler, mux *router.ServeMux) {
	mux.HandleFunc("GET /api/v1/account/session", sessionGet(h))
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

	isSignedIn := h.Session.IsSignedIn(ctx)

	return map[string]any{
		"isSignedIn":     isSignedIn,
		"isAwaitingTOTP": h.Session.IsAwaitingTOTP(ctx),
		"totpMethod":     h.Session.TOTPMethod(ctx),
		"isTOTPRequired": isSignedIn && config.TOTPRequired && !user.HasActivatedTOTP(),
	}
}
