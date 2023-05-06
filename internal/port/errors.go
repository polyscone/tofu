package port

import "github.com/polyscone/tofu/internal/pkg/errors"

var (
	ErrBadRequest     = errors.New("bad request")
	ErrUnauthorised   = errors.New("unauthorised")
	ErrMalformedInput = errors.New("malformed input")
	ErrInvalidInput   = errors.New("invalid input")
)
