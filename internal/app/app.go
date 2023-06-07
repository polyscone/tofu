package app

import "errors"

const (
	Name        = "Tofu"
	Description = "This is a base reference project for a hex architecture implementation in Go."
)

var (
	ErrBadRequest       = errors.New("bad request")
	ErrForbidden        = errors.New("forbidden")
	ErrUnauthorised     = errors.New("unauthorised")
	ErrMalformedInput   = errors.New("malformed input")
	ErrInvalidInput     = errors.New("invalid input")
	ErrConflictingInput = errors.New("conflicting input")
)
