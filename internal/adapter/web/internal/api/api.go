package api

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/polyscone/tofu/internal/adapter/web/internal/httputil"
	"github.com/polyscone/tofu/internal/adapter/web/internal/passport"
	"github.com/polyscone/tofu/internal/adapter/web/internal/sesskey"
	"github.com/polyscone/tofu/internal/adapter/web/internal/smtp"
	"github.com/polyscone/tofu/internal/adapter/web/internal/token"
	"github.com/polyscone/tofu/internal/pkg/command"
	"github.com/polyscone/tofu/internal/pkg/csrf"
	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/http/router"
	"github.com/polyscone/tofu/internal/pkg/logger"
	"github.com/polyscone/tofu/internal/pkg/session"
	"github.com/polyscone/tofu/internal/port"
	"github.com/polyscone/tofu/internal/port/account"
)

var (
	ErrBadJSON      = errors.New("bad json data")
	ErrExpectedJSON = errors.New("expected content-type application/json")
)

func init() {
	logger.AddSkipRule("api.writeError", logger.SkipFunc)
}

type API struct {
	bus      command.Bus
	sessions *session.Manager
	tokens   token.Repo
	mailer   smtp.Mailer
}

func New(bus command.Bus, sessions *session.Manager, tokens token.Repo, mailer smtp.Mailer) *API {
	return &API{
		bus:      bus,
		sessions: sessions,
		tokens:   tokens,
		mailer:   mailer,
	}
}

func (api *API) Routes() http.Handler {
	mux := router.NewServeMux()

	mux.Prefix("/api/v1", func(mux *router.ServeMux) {
		mux.Get("/csrf", api.csrfGet)

		mux.Prefix("/account", func(mux *router.ServeMux) {
			mux.Post("/register", api.accountRegisterPost)
			mux.Post("/activate", api.accountActivatePost)

			mux.Post("/totp", api.accountSetupTOTPPost)
			mux.Post("/totp/disable", api.accountDisableTOTPPost)
			mux.Post("/totp/verify", api.accountVerifyTOTPPost)
			mux.Put("/totp/recovery-codes", api.accountRegenerateRecoveryCodesPut)

			mux.Post("/login/password", api.accountLoginWithPasswordPost)
			mux.Post("/login/totp", api.accountLoginWithTOTPPost)
			mux.Post("/login/recovery-code", api.accountLoginWithRecoveryCodePost)
			mux.Post("/logout", api.accountLogoutPost)

			mux.Put("/password", api.accountChangePasswordPut)
			mux.Post("/password/reset", api.accountResetPasswordPost)
			mux.Put("/password/reset", api.accountResetPasswordPut)
		})
	})

	mux.NotFound(func(w http.ResponseWriter, r *http.Request) {
		writeError(w, r, errors.Tracef("%w: %v %v", httputil.ErrNotFound, r.Method, r.URL))
	})

	mux.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
		writeError(w, r, errors.Tracef("%w: %v %v", httputil.ErrMethodNotAllowed, r.Method, r.URL))
	})

	return mux
}

func (api *API) ErrorHandler(w http.ResponseWriter, r *http.Request, err error) {
	writeError(w, r, errors.Tracef(err))
}

func (api *API) passport(ctx context.Context) passport.Passport {
	if api.sessions.GetBool(ctx, sesskey.IsAwaitingTOTP) {
		return passport.Empty
	}

	userID := api.sessions.GetString(ctx, sesskey.UserID)
	cmd := account.FindAuthInfo{
		UserID: userID,
	}
	info, err := cmd.Execute(ctx, api.bus)
	if err != nil {
		return passport.Empty
	}

	return passport.New(info.Claims, info.Roles, info.Permissions)
}

func writeError(w http.ResponseWriter, r *http.Request, err error) bool {
	if err == nil {
		return false
	}

	httputil.LogError(r, err)

	var status int
	var displayOK bool

	switch {
	case errors.Is(err, ErrBadJSON):
		status, displayOK = http.StatusBadRequest, true

	case errors.Is(err, ErrExpectedJSON):
		status, displayOK = http.StatusUnsupportedMediaType, true

	default:
		status = httputil.ErrorStatus(err)

		switch {
		case errors.Is(err, port.ErrInvalidInput),
			errors.Is(err, port.ErrUnauthorised),
			errors.Is(err, csrf.ErrEmptyToken),
			errors.Is(err, csrf.ErrInvalidToken):

			displayOK = true
		}
	}

	w.Header().Set("content-type", "application/json")
	w.WriteHeader(status)

	detail := map[string]any{"error": strings.ToLower(http.StatusText(status))}
	if displayOK && 400 <= status && status <= 499 {
		detail["error"] = err.Error()

		if trace, ok := err.(errors.Trace); ok {
			fields := trace.Fields()

			if fields != nil {
				detail["fields"] = fields
			}
		}
	}

	if err := json.NewEncoder(w).Encode(detail); err != nil {
		httputil.LogError(r, err)
	}

	return true
}

func decodeJSON(r *http.Request, dst any) error {
	if !strings.HasPrefix(r.Header.Get("content-type"), "application/json") {
		return errors.Tracef(ErrExpectedJSON)
	}

	d := json.NewDecoder(r.Body)

	d.DisallowUnknownFields()

	if err := d.Decode(dst); err != nil {
		var syntaxErr *json.SyntaxError
		var unmarshalTypeErr *json.UnmarshalTypeError
		var invalidUnmarshalErr *json.InvalidUnmarshalError
		var maxBytesError *http.MaxBytesError

		switch {
		case errors.As(err, &invalidUnmarshalErr):
			panic(err)

		case errors.Is(err, io.EOF):
			return errors.Tracef(ErrBadJSON, "body must not be empty")

		case errors.As(err, &syntaxErr):
			return errors.Tracef(ErrBadJSON, "malformed JSON at offset %v", syntaxErr.Offset)

		case errors.Is(err, io.ErrUnexpectedEOF):
			return errors.Tracef(ErrBadJSON, "malformed JSON")

		case errors.As(err, &unmarshalTypeErr):
			return errors.Tracef(ErrBadJSON, "invalid value for %q at offset %v", unmarshalTypeErr.Field, unmarshalTypeErr.Offset)

		case strings.HasPrefix(err.Error(), "json: unknown field "):
			fieldName := strings.TrimPrefix(err.Error(), "json: unknown field ")

			return errors.Tracef(ErrBadJSON, "unknown field %v", fieldName)

		case errors.As(err, &maxBytesError):
			return errors.Tracef(ErrBadJSON, "request body must be no larger than %v bytes", maxBytesError.Limit)

		default:
			return errors.Tracef(ErrBadJSON, err)
		}
	}

	if err := d.Decode(&struct{}{}); err != io.EOF {
		return errors.Tracef(ErrBadJSON, "unexpected additional JSON")
	}

	return nil
}

func writeJSON(w http.ResponseWriter, r *http.Request, data any) bool {
	w.Header().Set("content-type", "application/json")

	return !writeError(w, r, errors.Tracef(json.NewEncoder(w).Encode(data)))
}
