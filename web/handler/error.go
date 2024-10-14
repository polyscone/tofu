package handler

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/polyscone/tofu/app"
	"github.com/polyscone/tofu/app/account"
	"github.com/polyscone/tofu/internal/csrf"
	"github.com/polyscone/tofu/internal/httpx"
	"github.com/polyscone/tofu/internal/human"
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

	case errors.Is(err, app.ErrUnauthorised):
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

func ErrorMessage(err error) string {
	switch {
	case errors.Is(err, httpx.ErrNotFound),
		errors.Is(err, app.ErrNotFound):

		return "The resource you were looking for could not be found."

	case errors.Is(err, httpx.ErrMethodNotAllowed):
		return "Method not allowed."

	case errors.Is(err, httpx.ErrForbidden),
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
		errors.Is(err, app.ErrConflict):

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

	case errors.Is(err, app.ErrRepoLogin):
		return "Could not connect to the datasource."

	default:
		var maxBytesError *http.MaxBytesError
		if errors.As(err, &maxBytesError) {
			return fmt.Sprintf("Your request must be no larger than %v.", human.SizeSI(uint64(maxBytesError.Limit)))
		}

		return "An error has occurred."
	}
}
