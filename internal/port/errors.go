package port

import "github.com/polyscone/tofu/internal/pkg/errors"

var (
	ErrBadRequest   = errors.New("bad request")
	ErrUnauthorised = errors.New("unauthorised")
	ErrInvalidInput = errors.New("invalid input")
)
