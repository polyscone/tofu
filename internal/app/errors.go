package app

import "github.com/polyscone/tofu/internal/pkg/errors"

var (
	ErrBadRequest   = errors.New("bad request")
	ErrUnauthorized = errors.New("unauthorised")
	ErrInvalidInput = errors.New("invalid input")
)
