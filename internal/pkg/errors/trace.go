package errors

import (
	"errors"
	"fmt"
	"runtime"
	"strings"
)

// Frame holds all of the information pertaining to the point in a program where
// an error was traced.
type Frame struct {
	PC       uintptr `json:"-"`
	File     string  `json:"-"`
	Line     int     `json:"-"`
	FuncName string  `json:"-"`
}

// Trace represents an error that has had its location in the program recorded.
type Trace struct {
	Frame  Frame `json:"-"`
	Kind   error `json:"-"`
	Err    error `json:"-"`
	fields Map   `json:"-"`
}

// String implements the fmt.Stringer interface.
func (t Trace) String() string {
	var sb strings.Builder

	fprintln(&sb, t, "")

	return strings.TrimRight(sb.String(), "\n\r")
}

// Error implements the error interface.
func (t Trace) Error() string {
	if t.Err == nil {
		return "<nil>"
	}

	s := t.Err.Error()

	if t.fields != nil {
		s += ": " + t.fields.String()
	}

	return s
}

// Is implements checks for use with errors.IS
func (t Trace) Is(target error) bool {
	return t.Kind == target
}

// Unwrap returns the error that has been traced.
func (t Trace) Unwrap() error {
	return t.Err
}

// Fields returns an error map which represents a collection of errors
// keyed by name.
func (t Trace) Fields() Map {
	var fields Map

	err := error(t)
	for err != nil {
		if trace, ok := err.(Trace); ok && trace.fields != nil {
			for key, err := range trace.fields {
				fields.Set(key, err)
			}
		}

		err = errors.Unwrap(err)
	}

	return fields
}

// MarshalJSON implements the json.Marshaler interface.
func (t Trace) MarshalJSON() ([]byte, error) {
	return []byte(sprintJSON(t)), nil
}

// Tracef returns a new error that also holds information about where the error
// occurred in the program.
//
// There are multiple ways to use this function based on the argument types.
// The first argument must always be either an error or a string.
// When the first argument is an error the second argument must also always be
// either an error or a string.
//
// When the first argument is an error then it is always checked for nil.
// If it is nil then no error is created, no work is done, and nil is returned.
//
// For (error), that error is used as both the kind of error in errors.Is checks
// and as the error message.
// If error is nil then nil is returned and no error is created.
//
// For (string, ...any), the string is used as a format string to always
// create a new error.
//
// For (error, string, ...any), error is used as the kind of error in
// errors.Is checks, and the string is used as a format string to always create
// a new error.
// If error is nil then nil is returned and no error is created.
//
// For (error, error), the first error is used as the error message and the
// second is used as the kind of error in errors.Is checks.
// In this case the second error must not be nil.
// If the first error is nil then nil is returned and no error is created.
//
// For (error, error, string, ...any), the first error is used only to check for
// nil, and the second is used as the kind of error in errors.Is checks.
// In this case the second error must not be nil.
// The string is used as a format string to create a new error when the first
// error is not nil.
// If the first error is nil then nil is returned and no error is created.
func Tracef(errFormat any, a ...any) error {
	return tracef(2, errFormat, a...)
}

func tracef(skip int, errFormat any, a ...any) error {
	if errFormat == nil {
		return nil
	}

	var kind error
	if v, ok := errFormat.(error); ok {
		if len(a) != 0 {
			switch err := a[0].(type) {
			case nil:
				panic("want second arg to be non-nil error or string value")

			case error:
				v = err
				a = a[1:]
			}
		}

		if trace, ok := v.(Trace); ok {
			kind = trace.Kind
		} else {
			kind = v
		}

		if len(a) != 0 {
			errFormat = a[0]
			a = a[1:]
		}
	}

	var err error
	switch value := errFormat.(type) {
	case error:
		err = value

	case string:
		err = fmt.Errorf(value, a...)

	default:
		panic("want non-nil error or string value")
	}

	trace := Trace{
		Kind: kind,
		Err:  err,
	}

	if pc, file, line, ok := runtime.Caller(skip); ok {
		trace.Frame = Frame{
			PC:       pc,
			File:     file,
			Line:     line,
			FuncName: runtime.FuncForPC(pc).Name(),
		}
	}

	return trace
}

// StackTrace generates a slice of Trace objects that represent the trace of an
// error throughout the program.
func StackTrace(err error) []Trace {
	var stack []Trace

	var prev *Frame
	for err != nil {
		if trace, ok := err.(Trace); ok {
			if prev == nil || *prev != trace.Frame {
				stack = append(stack, trace)
			}

			prev = &trace.Frame
		}

		err = errors.Unwrap(err)
	}

	// Reverse the stack
	for i := len(stack)/2 - 1; i >= 0; i-- {
		opp := len(stack) - 1 - i

		stack[i], stack[opp] = stack[opp], stack[i]
	}

	return stack
}
