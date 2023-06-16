package app

import (
	"errors"
	"time"
)

const (
	Name              = "App"
	Description       = "This is a boilerplate project for applications written in Go."
	SignInThrottleTTL = 24 * time.Hour
)

var (
	ErrBadRequest       = errors.New("bad request")
	ErrForbidden        = errors.New("forbidden")
	ErrUnauthorised     = errors.New("unauthorised")
	ErrMalformedInput   = errors.New("malformed input")
	ErrInvalidInput     = errors.New("invalid input")
	ErrConflictingInput = errors.New("conflicting input")
)
