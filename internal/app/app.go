package app

import (
	"errors"
	"time"
)

const (
	Name        = "App"
	ShortName   = "App"
	Description = "This is a boilerplate project for applications written in Go."
	ThemeColour = "hsl(170, 45%, 30%)"

	SessionTTL        = 2 * time.Hour
	SignInThrottleTTL = 30 * time.Minute
)

var (
	ErrBadRequest       = errors.New("bad request")
	ErrNotFound         = errors.New("not found")
	ErrUnauthorised     = errors.New("unauthorised")
	ErrForbidden        = errors.New("forbidden")
	ErrMalformedInput   = errors.New("malformed input")
	ErrInvalidInput     = errors.New("invalid input")
	ErrConflictingInput = errors.New("conflicting input")
)
