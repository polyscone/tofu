package errors

// Must accepts any function call that returns two values, T and error.
// If error is not nil then it panics, otherwise it returns the value T.
func Must[T any](value T, err error) T {
	if err != nil {
		panic(err)
	}

	return value
}

// MustOK accepts any function call that returns two values, T and bool.
// If bool is false then it panics, otherwise it returns the value T.
func MustOK[T any](value T, ok bool) T {
	if !ok {
		panic("ok value must be true; got false")
	}

	return value
}
