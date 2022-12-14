package errors

import "errors"

// As wraps the standard library's errors.As function.
func As(err error, target any) bool {
	return errors.As(err, target)
}

// Is wraps the standard library's errors.Is function.
func Is(err, target error) bool {
	return errors.Is(err, target)
}

// Unwrap wraps the standard library's errors.Unwrap function.
func Unwrap(err error) error {
	return errors.Unwrap(err)
}

// New wraps the standard library's errors.New function.
func New(text string) error {
	return errors.New(text)
}
