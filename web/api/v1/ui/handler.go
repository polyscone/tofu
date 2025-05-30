package ui

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"slices"
	"strings"

	"github.com/polyscone/tofu/app"
	"github.com/polyscone/tofu/app/account"
	"github.com/polyscone/tofu/internal/csrf"
	"github.com/polyscone/tofu/internal/errsx"
	"github.com/polyscone/tofu/internal/httpx/router"
	"github.com/polyscone/tofu/internal/i18n"
	"github.com/polyscone/tofu/web/guard"
	"github.com/polyscone/tofu/web/handler"
)

var publicErrors = []error{
	account.ErrSignInThrottled,
	app.ErrInvalidInput,
	app.ErrMalformedInput,
	csrf.ErrEmptyToken,
	csrf.ErrInvalidToken,
}

type PredicateFunc func(p guard.Passport) bool

type Handler struct {
	*handler.Handler
	i18nRuntime i18n.Runtime
}

func NewHandler(base *handler.Handler, mux *router.ServeMux) *Handler {
	i18nRuntimeWrapper := handler.NewI18nRuntimeWrapper(mux)

	return &Handler{
		Handler:     base,
		i18nRuntime: i18nRuntimeWrapper(i18n.DefaultJSRuntime),
	}
}

func (h *Handler) T(ctx context.Context, message i18n.Message) string {
	locale := h.Locale(ctx)
	res, err := i18n.T(h.i18nRuntime, locale, message)
	if err != nil {
		logger := h.Logger(ctx)

		logger.Error("api v1 handler: i18n T", "error", err)
	}

	return res.AsString().Value
}

func (h *Handler) JSON(w http.ResponseWriter, r *http.Request, status int, data any) {
	w.Header().Set("content-type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		ctx := r.Context()

		h.Logger(ctx).Error("write JSON response", "error", err)
	}
}

func (h *Handler) RawJSON(w http.ResponseWriter, r *http.Request, status int, data []byte) {
	w.Header().Set("content-type", "application/json")
	w.WriteHeader(status)

	if _, err := w.Write(data); err != nil {
		ctx := r.Context()

		h.Logger(ctx).Error("write raw JSON response", "error", err)
	}
}

func (h *Handler) ErrorJSON(w http.ResponseWriter, r *http.Request, msg string, err error) {
	ctx := r.Context()
	logger := h.Logger(ctx)

	logger.Error(msg, "error", err)

	status := handler.ErrorStatus(err)
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
		detail["error"] = h.T(ctx, handler.ErrorMessage(err))

		var errs errsx.Map
		if errors.As(err, &errs) {
			detail["fields"] = errs
		}

		var throttled *account.SignInThrottleError
		if errors.As(err, &throttled) {
			detail["inLast"] = h.T(ctx, i18n.M("api:account.sign_in.throttled.in_last", "in_last", throttled.InLast))
			detail["unlockIn"] = h.T(ctx, i18n.M("api:account.sign_in.throttled.unlock_in", "unlock_in", throttled.UnlockIn))
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

func (h *Handler) RequireSignIn(w http.ResponseWriter, r *http.Request) bool {
	ctx := r.Context()
	if h.Session.IsSignedIn(ctx) {
		return false
	}

	h.ErrorJSON(w, r, "require sign in", app.ErrUnauthorized)

	return true
}

func (h *Handler) Forbidden(w http.ResponseWriter, r *http.Request, allowed PredicateFunc) bool {
	ctx := r.Context()
	passport := h.Passport(ctx)

	if allowed(passport) {
		return false
	}

	h.ErrorJSON(w, r, "forbidden", app.ErrForbidden)

	return true
}
