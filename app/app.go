package app

import (
	"errors"
	"time"

	"github.com/polyscone/tofu/errsx"
)

const (
	Name        = "App"
	ShortName   = "App"
	Description = "This is a boilerplate project for applications written in Go."
	ThemeColour = "hsl(170, 45%, 30%)"

	SessionTTL        = 2 * time.Hour
	SignInThrottleTTL = 30 * time.Minute
)

var BasePath = ""

var (
	ErrBadRequest     = errors.New("bad request")
	ErrNotFound       = errors.New("not found")
	ErrUnauthorised   = errors.New("unauthorised")
	ErrForbidden      = errors.New("forbidden")
	ErrMalformedInput = errors.New("malformed input")
	ErrInvalidInput   = errors.New("invalid input")
	ErrConflict       = errors.New("conflict")
	ErrRepoLogin      = errors.New("login")
)

type ConflictError struct {
	errsx.Map
}

func (c ConflictError) Error() string {
	return c.Map.String()
}

func (c ConflictError) Unwrap() error {
	return c.Map
}
