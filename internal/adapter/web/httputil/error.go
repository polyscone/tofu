package httputil

import (
	"errors"
	"net/http"

	"github.com/polyscone/tofu/internal/app"
	"github.com/polyscone/tofu/internal/app/account"
	"github.com/polyscone/tofu/internal/pkg/csrf"
	"github.com/polyscone/tofu/internal/pkg/rate"
	"github.com/polyscone/tofu/internal/repository"
)

const StatusClientClosedRequest = 499

var (
	ErrNotFound            = errors.New("not found")
	ErrForbidden           = errors.New("forbidden")
	ErrMethodNotAllowed    = errors.New("method not allowed")
	ErrInternalServerError = errors.New("internal server error")
)

func ErrorStatus(err error) int {
	switch {
	case errors.Is(err, repository.ErrLogin):
		return http.StatusBadGateway

	case errors.Is(err, http.ErrHandlerTimeout):
		return http.StatusGatewayTimeout

	case errors.Is(err, account.ErrGoogleSignUpDisabled),
		errors.Is(err, app.ErrMalformedInput),
		errors.Is(err, app.ErrInvalidInput),
		errors.Is(err, app.ErrConflictingInput),
		errors.Is(err, app.ErrBadRequest),
		errors.Is(err, csrf.ErrEmptyToken),
		errors.Is(err, csrf.ErrInvalidToken),
		errors.Is(err, ErrBadJSON):

		return http.StatusBadRequest

	case errors.Is(err, app.ErrNotFound),
		errors.Is(err, ErrNotFound):

		return http.StatusNotFound

	case errors.Is(err, account.ErrNotVerified),
		errors.Is(err, account.ErrNotActivated),
		errors.Is(err, account.ErrSuspended),
		errors.Is(err, app.ErrForbidden),
		errors.Is(err, ErrForbidden):

		return http.StatusForbidden

	case errors.Is(err, app.ErrUnauthorised):
		return http.StatusUnauthorized

	case errors.Is(err, ErrMethodNotAllowed):
		return http.StatusMethodNotAllowed

	case errors.Is(err, ErrExpectedJSON):
		return http.StatusUnsupportedMediaType

	case errors.Is(err, rate.ErrInsufficientTokens),
		errors.Is(err, account.ErrSignInThrottled):

		return http.StatusTooManyRequests

	default:
		var maxBytesError *http.MaxBytesError

		if errors.As(err, &maxBytesError) {
			return http.StatusRequestEntityTooLarge
		}
	}

	return http.StatusInternalServerError
}

func ErrorMessage(err error) string {
	switch {
	case errors.Is(err, ErrNotFound),
		errors.Is(err, app.ErrNotFound):

		return "The page you were looking for could not be found."

	case errors.Is(err, ErrMethodNotAllowed):
		return "Method not allowed."

	case errors.Is(err, ErrForbidden),
		errors.Is(err, app.ErrForbidden):

		return "You do not have sufficient permissions to access this resource."

	case errors.Is(err, http.ErrHandlerTimeout):
		return "The server took too long to respond."

	case errors.Is(err, account.ErrNotVerified):
		return "This account is not verified."

	case errors.Is(err, account.ErrNotActivated):
		return "This account is not activated."

	case errors.Is(err, account.ErrSuspended):
		return "This account has been suspended."

	case errors.Is(err, app.ErrUnauthorised):
		return "You do not have permission to access this resource."

	case errors.Is(err, app.ErrMalformedInput),
		errors.Is(err, app.ErrInvalidInput),
		errors.Is(err, app.ErrConflictingInput):

		if errors.Is(err, app.ErrMalformedInput) {
			return "Malformed input."
		} else {
			return "Invalid input."
		}

	case errors.Is(err, csrf.ErrEmptyToken):
		return "Empty CSRF token."

	case errors.Is(err, csrf.ErrInvalidToken):
		return "Invalid CSRF token."

	case errors.Is(err, rate.ErrInsufficientTokens),
		errors.Is(err, account.ErrSignInThrottled):

		return "You have made too many consecutive requests. Please try again later."

	case errors.Is(err, repository.ErrLogin):
		return "Could not connect to the datasource."

	default:
		return "An error has occurred."
	}
}
