package httpx

import "errors"

const StatusClientClosedRequest = 499

var (
	ErrNotFound            = errors.New("not found")
	ErrForbidden           = errors.New("forbidden")
	ErrMethodNotAllowed    = errors.New("method not allowed")
	ErrInternalServerError = errors.New("internal server error")
)
