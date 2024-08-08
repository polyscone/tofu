package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"

	"github.com/polyscone/tofu/app"
	"github.com/polyscone/tofu/app/account"
	"github.com/polyscone/tofu/csrf"
	"github.com/polyscone/tofu/errsx"
	"github.com/polyscone/tofu/httpx"
	"github.com/polyscone/tofu/human"
	"github.com/polyscone/tofu/web/handler"
	"github.com/polyscone/tofu/web/sess"
)

var publicErrors = []error{
	account.ErrSignInThrottled,
	app.ErrInvalidInput,
	app.ErrMalformedInput,
	csrf.ErrEmptyToken,
	csrf.ErrInvalidToken,
}

type Handler struct {
	*handler.Handler
}

func NewHandler(base *handler.Handler) *Handler {
	return &Handler{Handler: base}
}

func (h *Handler) JSON(w http.ResponseWriter, r *http.Request, status int, data any) {
	w.Header().Set("content-type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		ctx := r.Context()

		h.Logger(ctx).Error("write JSON response", "error", err)
	}
}

func (h *Handler) RawJSON(w http.ResponseWriter, r *http.Request, status int, data string) {
	w.Header().Set("content-type", "application/json")
	w.WriteHeader(status)

	if _, err := fmt.Fprint(w, data); err != nil {
		ctx := r.Context()

		h.Logger(ctx).Error("write raw JSON response", "error", err)
	}
}

func (h *Handler) ErrorJSON(w http.ResponseWriter, r *http.Request, msg string, err error) {
	ctx := r.Context()
	logger := h.Logger(ctx)

	logger.Error(msg, "error", err)

	status := httpx.ErrorStatus(err)
	isPublic := slices.ContainsFunc(publicErrors, func(el error) bool {
		return errors.Is(err, el)
	})

	if status == http.StatusTooManyRequests {
		// If a client is hitting a rate limit we set the connection header to
		// close which will trigger the standard library's HTTP server to close
		// the connection after the response is sent
		//
		// Doing this means the client needs to go through the handshake process
		// again to make a new connection the next time, which should help to slow
		// down additional requests for clients that keep on hitting the limit
		w.Header().Set("connection", "close")
	}

	w.Header().Set("content-type", "application/json")
	w.WriteHeader(status)

	detail := map[string]any{"error": strings.ToLower(http.StatusText(status))}
	if isPublic && 400 <= status && status <= 499 {
		detail["error"] = httpx.ErrorMessage(err)

		var errs errsx.Map
		if errors.As(err, &errs) {
			detail["fields"] = errs
		}

		var throttled *account.SignInThrottleError
		if errors.As(err, &throttled) {
			detail["inLast"] = human.Duration(throttled.InLast)
			detail["unlockIn"] = human.Duration(throttled.UnlockIn)
		}

		switch {
		case errors.Is(err, csrf.ErrEmptyToken):
			detail["csrf"] = "empty"

		case errors.Is(err, csrf.ErrInvalidToken):
			detail["csrf"] = "invalid"
		}
	}

	if err := json.NewEncoder(w).Encode(detail); err != nil {
		logger.Error("write error JSON response", "error", err)
	}
}

func (h *Handler) RequireSignIn(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		isSignedIn := h.Sessions.GetBool(ctx, sess.IsSignedIn)

		if !isSignedIn {
			h.ErrorJSON(w, r, "require sign in", app.ErrUnauthorised)

			return
		}

		next(w, r)
	}
}
