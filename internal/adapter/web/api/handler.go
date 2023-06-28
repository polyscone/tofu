package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/polyscone/tofu/internal/adapter/web/handler"
	"github.com/polyscone/tofu/internal/adapter/web/httputil"
	"github.com/polyscone/tofu/internal/app"
	"github.com/polyscone/tofu/internal/app/account"
	"github.com/polyscone/tofu/internal/pkg/errsx"
	"golang.org/x/exp/slices"
)

var publicErrors = []error{
	account.ErrSignInThrottled,
	app.ErrInvalidInput,
	app.ErrMalformedInput,
}

type Handler struct {
	*handler.Handler
}

func NewHandler(base *handler.Handler) *Handler {
	return &Handler{Handler: base}
}

func (h *Handler) JSON(w http.ResponseWriter, r *http.Request, data any) {
	ctx := r.Context()

	w.Header().Set("content-type", "application/json")

	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.Logger(ctx).Error("write JSON response", "error", err)
	}
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
	}

	if err := json.NewEncoder(w).Encode(detail); err != nil {
		h.Logger(ctx).Error("write error JSON response", "error", err)
	}
}
