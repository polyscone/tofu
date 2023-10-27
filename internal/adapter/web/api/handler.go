package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"

	"github.com/polyscone/tofu/internal/adapter/web/handler"
	"github.com/polyscone/tofu/internal/adapter/web/httputil"
	"github.com/polyscone/tofu/internal/adapter/web/sess"
	"github.com/polyscone/tofu/internal/app"
	"github.com/polyscone/tofu/internal/app/account"
	"github.com/polyscone/tofu/internal/pkg/csrf"
	"github.com/polyscone/tofu/internal/pkg/errsx"
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
	ctx := r.Context()

	w.Header().Set("content-type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.Logger(ctx).Error("write JSON response", "error", err)
	}
}

func (h *Handler) RawJSON(w http.ResponseWriter, r *http.Request, status int, data string) {
	w.Header().Set("content-type", "application/json")
	w.WriteHeader(status)

	fmt.Fprint(w, data)
}

func (h *Handler) ErrorJSON(w http.ResponseWriter, r *http.Request, msg string, err error) {
	ctx := r.Context()

	h.Logger(ctx).Error(msg, "error", err)

	status := httputil.ErrorStatus(err)
	isPublic := slices.ContainsFunc(publicErrors, func(el error) bool {
		return errors.Is(err, el)
	})

	w.Header().Set("content-type", "application/json")
	w.WriteHeader(status)

	detail := map[string]any{"error": strings.ToLower(http.StatusText(status))}
	if isPublic && 400 <= status && status <= 499 {
		detail["error"] = err.Error()

		var errs errsx.Map
		if errors.As(err, &errs) {
			detail["fields"] = errs
		}

		var throttled *account.SignInThrottleError
		if errors.As(err, &throttled) {
			detail["inLast"] = throttled.InLast
			detail["unlockIn"] = throttled.UnlockIn
		}

		switch {
		case errors.Is(err, csrf.ErrEmptyToken):
			detail["csrf"] = "empty"

		case errors.Is(err, csrf.ErrInvalidToken):
			detail["csrf"] = "invalid"
		}
	}

	if err := json.NewEncoder(w).Encode(detail); err != nil {
		h.Logger(ctx).Error("write error JSON response", "error", err)
	}
}

func (h *Handler) RequireSignIn(w http.ResponseWriter, r *http.Request) bool {
	ctx := r.Context()
	isSignedIn := h.Sessions.GetBool(ctx, sess.IsSignedIn)

	if !isSignedIn {
		h.ErrorJSON(w, r, "require sign in", app.ErrUnauthorised)

		return false
	}

	config := h.Config(ctx)
	user := h.User(ctx)

	if config.TOTPRequired && !user.HasActivatedTOTP() {
		h.ErrorJSON(w, r, "require TOTP", app.ErrUnauthorised)

		return false
	}

	return true
}
