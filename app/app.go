package app

import (
	"errors"
	"time"

	"github.com/polyscone/tofu/config"
	"github.com/polyscone/tofu/internal/errsx"
)

func init() {
	errsx.Must0(config.LoadI18nLocales())
}

const (
	Name        = "App"
	ShortName   = "App"
	Description = "This is a boilerplate project for applications written in Go."
	ThemeColor  = "hsl(170, 45%, 30%)"

	SessionTTL        = 2 * time.Hour
	SignInThrottleTTL = 30 * time.Minute
)

var BasePath = ""

var (
	ErrBadRequest     = errors.New("bad request")
	ErrNotFound       = errors.New("not found")
	ErrUnauthorized   = errors.New("unauthorized")
	ErrForbidden      = errors.New("forbidden")
	ErrMalformedInput = errors.New("malformed input")
	ErrInvalidInput   = errors.New("invalid input")
	ErrConflict       = errors.New("conflict")
	ErrRepoLogin      = errors.New("login")
	ErrLoopDetected   = errors.New("loop detected")
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
