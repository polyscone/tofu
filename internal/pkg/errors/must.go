package errors

// Must0 accepts any function call that returns an error.
// If error is not nil then it panics.
func Must0(err error) {
	if err != nil {
		panic(err)
	}
}

// Must1 accepts any function call that returns two values, T and error.
// If error is not nil then it panics, otherwise it returns the value T.
func Must1[T any](value T, err error) T {
	if err != nil {
		panic(err)
	}

	return value
}

// Must is an alias for Must1.
func Must[T any](value T, err error) T {
	return Must1(value, err)
}

// MustOK accepts any function call that returns two values, T and bool.
// If bool is false then it panics, otherwise it returns the value T.
func MustOK[T any](value T, ok bool) T {
	if !ok {
		panic("ok value must be true; got false")
	}

	return value
}
