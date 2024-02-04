package api

import (
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"slices"
	"strings"

	"github.com/polyscone/tofu/internal/adapter/web/handler"
	"github.com/polyscone/tofu/internal/adapter/web/httputil"
	"github.com/polyscone/tofu/internal/adapter/web/sess"
	"github.com/polyscone/tofu/internal/app"
	"github.com/polyscone/tofu/internal/app/account"
	"github.com/polyscone/tofu/internal/pkg/csrf"
	"github.com/polyscone/tofu/internal/pkg/dev"
	"github.com/polyscone/tofu/internal/pkg/errsx"
	"github.com/polyscone/tofu/internal/pkg/fstack"
	"github.com/polyscone/tofu/internal/pkg/human"
	"github.com/polyscone/tofu/internal/repository"
)

//go:embed "all:template"
var files embed.FS

const templateDir = "template"

var templateFiles = fstack.New(dev.RelDirFS(templateDir), errsx.Must(fs.Sub(files, templateDir)))

var publicErrors = []error{
	account.ErrSignInThrottled,
	app.ErrInvalidInput,
	app.ErrMalformedInput,
	csrf.ErrEmptyToken,
	csrf.ErrInvalidToken,
}

type Handler struct {
	*handler.Handler
	JavaScript *handler.Renderer
}

func NewHandler(base *handler.Handler) *Handler {
	templatePaths := func(view string) []string {
		return []string{view}
	}

	funcs := handler.NewTemplateFuncs(nil)

	return &Handler{
		Handler:    base,
		JavaScript: handler.NewRenderer(base, templateFiles, templatePaths, funcs, "application/javascript"),
	}
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
	if !isPublic {
		var inputError repository.InputError
		if errors.As(err, &inputError) {
			isPublic = true
		}
	}

	w.Header().Set("content-type", "application/json")
	w.WriteHeader(status)

	detail := map[string]any{"error": strings.ToLower(http.StatusText(status))}
	if isPublic && 400 <= status && status <= 499 {
		detail["error"] = httputil.ErrorMessage(err)

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
		h.Logger(ctx).Error("write error JSON response", "error", err)
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
