package app

import "errors"

const (
	Name        = "App Name"
	Description = "This is a base reference project for a hex architecture implementation in Go."
)

var (
	ErrBadRequest     = errors.New("bad request")
	ErrUnauthorised   = errors.New("unauthorised")
	ErrMalformedInput = errors.New("malformed input")
	ErrInvalidInput   = errors.New("invalid input")
)
