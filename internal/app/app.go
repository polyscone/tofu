package app

import "errors"

const (
	Name        = "App"
	Description = "This is a boilerplate project for applications written in Go."
)

var (
	ErrBadRequest       = errors.New("bad request")
	ErrForbidden        = errors.New("forbidden")
	ErrUnauthorised     = errors.New("unauthorised")
	ErrMalformedInput   = errors.New("malformed input")
	ErrInvalidInput     = errors.New("invalid input")
	ErrConflictingInput = errors.New("conflicting input")
)
