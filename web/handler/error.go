package handler

import (
	"errors"
	"net/http"

	"github.com/polyscone/tofu/app"
	"github.com/polyscone/tofu/app/account"
	"github.com/polyscone/tofu/internal/csrf"
	"github.com/polyscone/tofu/internal/httpx"
	"github.com/polyscone/tofu/internal/human"
	"github.com/polyscone/tofu/internal/i18n"
	"github.com/polyscone/tofu/internal/rate"
)

func ErrorStatus(err error) int {
	switch {
	case errors.Is(err, app.ErrRepoLogin):
		return http.StatusBadGateway

	case errors.Is(err, http.ErrHandlerTimeout):
		return http.StatusGatewayTimeout

	case errors.Is(err, rate.ErrInsufficientTokens),
		errors.Is(err, account.ErrSignInThrottled):

		return http.StatusTooManyRequests

	case errors.Is(err, account.ErrAuth),
		errors.Is(err, account.ErrGoogleSignUpDisabled),
		errors.Is(err, app.ErrMalformedInput),
		errors.Is(err, app.ErrInvalidInput),
		errors.Is(err, app.ErrConflict),
		errors.Is(err, app.ErrBadRequest),
		errors.Is(err, csrf.ErrEmptyToken),
		errors.Is(err, csrf.ErrInvalidToken),
		errors.Is(err, httpx.ErrBadJSON):

		return http.StatusBadRequest

	case errors.Is(err, app.ErrNotFound),
		errors.Is(err, httpx.ErrNotFound):

		return http.StatusNotFound

	case errors.Is(err, account.ErrNotVerified),
		errors.Is(err, account.ErrNotActivated),
		errors.Is(err, account.ErrSuspended),
		errors.Is(err, app.ErrForbidden),
		errors.Is(err, httpx.ErrForbidden):

		return http.StatusForbidden

	case errors.Is(err, app.ErrUnauthorized):
		return http.StatusUnauthorized

	case errors.Is(err, httpx.ErrMethodNotAllowed):
		return http.StatusMethodNotAllowed

	case errors.Is(err, httpx.ErrExpectedJSON):
		return http.StatusUnsupportedMediaType

	default:
		var maxBytesError *http.MaxBytesError
		if errors.As(err, &maxBytesError) {
			return http.StatusRequestEntityTooLarge
		}
	}

	return http.StatusInternalServerError
}

func ErrorMessage(err error) i18n.Message {
	switch {
	case errors.Is(err, httpx.ErrNotFound),
		errors.Is(err, app.ErrNotFound):

		return i18n.M("web.error.not_found")

	case errors.Is(err, httpx.ErrMethodNotAllowed):
		return i18n.M("web.error.http_method_not_allowed")

	case errors.Is(err, httpx.ErrForbidden),
		errors.Is(err, app.ErrForbidden):

		return i18n.M("web.error.forbidden")

	case errors.Is(err, http.ErrHandlerTimeout):
		return i18n.M("web.error.handler_timeout")

	case errors.Is(err, account.ErrNotVerified):
		return i18n.M("web.error.account_not_verified")

	case errors.Is(err, account.ErrNotActivated):
		return i18n.M("web.error.account_not_activated")

	case errors.Is(err, account.ErrSuspended):
		return i18n.M("web.error.account_suspended")

	case errors.Is(err, app.ErrUnauthorized):
		return i18n.M("web.error.unauthorized")

	case errors.Is(err, app.ErrMalformedInput):
		return i18n.M("web.error.malformed_input")

	case errors.Is(err, app.ErrInvalidInput),
		errors.Is(err, app.ErrConflict):

		return i18n.M("web.error.invalid_input")

	case errors.Is(err, csrf.ErrEmptyToken):
		return i18n.M("web.error.empty_csrf_token")

	case errors.Is(err, csrf.ErrInvalidToken):
		return i18n.M("web.error.invalid_csrf_token")

	case errors.Is(err, rate.ErrInsufficientTokens),
		errors.Is(err, account.ErrSignInThrottled):

		return i18n.M("web.error.too_many_requests")

	case errors.Is(err, app.ErrRepoLogin):
		return i18n.M("web.error.repo_login")

	default:
		var maxBytesError *http.MaxBytesError
		if errors.As(err, &maxBytesError) {
			return i18n.M("web.error.request_too_large", "max_size", human.SizeSI(maxBytesError.Limit))
		}

		return i18n.M("web.error.generic")
	}
}
