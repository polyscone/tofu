package httputil

import (
	"errors"
	"net/http"

	"github.com/polyscone/tofu/internal/app"
	"github.com/polyscone/tofu/internal/pkg/csrf"
	"github.com/polyscone/tofu/internal/pkg/rate"
)

var (
	ErrNotFound            = errors.New("not found")
	ErrForbidden           = errors.New("forbidden")
	ErrMethodNotAllowed    = errors.New("method not allowed")
	ErrInternalServerError = errors.New("internal server error")
)

func ErrorStatus(err error) int {
	switch {
	case errors.Is(err, http.ErrHandlerTimeout):
		return http.StatusGatewayTimeout

	case errors.Is(err, app.ErrMalformedInput),
		errors.Is(err, app.ErrInvalidInput),
		errors.Is(err, app.ErrBadRequest),
		errors.Is(err, csrf.ErrEmptyToken),
		errors.Is(err, csrf.ErrInvalidToken):

		return http.StatusBadRequest

	case errors.Is(err, ErrNotFound):
		return http.StatusNotFound

	case errors.Is(err, ErrForbidden):
		return http.StatusForbidden

	case errors.Is(err, app.ErrUnauthorised):
		return http.StatusUnauthorized

	case errors.Is(err, ErrMethodNotAllowed):
		return http.StatusMethodNotAllowed

	case errors.Is(err, rate.ErrInsufficientTokens):
		return http.StatusTooManyRequests

	default:
		var maxBytesError *http.MaxBytesError

		if errors.As(err, &maxBytesError) {
			return http.StatusRequestEntityTooLarge
		}
	}

	return http.StatusInternalServerError
}
