package httputil

import (
	"errors"
	"net/http"

	"github.com/polyscone/tofu/internal/pkg/csrf"
	"github.com/polyscone/tofu/internal/pkg/rate"
	"github.com/polyscone/tofu/internal/port"
)

var (
	ErrNotFound         = errors.New("not found")
	ErrMethodNotAllowed = errors.New("method not allowed")
)

func ErrorStatus(err error) int {
	switch {
	case errors.Is(err, http.ErrHandlerTimeout):
		return http.StatusGatewayTimeout

	case errors.Is(err, port.ErrMalformedInput),
		errors.Is(err, port.ErrInvalidInput),
		errors.Is(err, port.ErrBadRequest),
		errors.Is(err, csrf.ErrEmptyToken),
		errors.Is(err, csrf.ErrInvalidToken):

		return http.StatusBadRequest

	case errors.Is(err, ErrNotFound):
		return http.StatusNotFound

	case errors.Is(err, port.ErrUnauthorised):
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
