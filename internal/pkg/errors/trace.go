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
// The given format string and format arguments are used to describe the
// traced error.
// If the first argument passed is an error then that error will be used as the
// kind of error in checks using errors.Is and the format string is considered
// to start from the second argument.
// In the case where an error is provided as the first argument the format
// string is optional.
func Tracef(format any, a ...any) error {
	return tracef(2, format, a...)
}

func tracef(skip int, format any, a ...any) error {
	if format == nil {
		return nil
	}

	var kind error
	if v, ok := format.(error); ok {
		if trace, ok := v.(Trace); ok {
			kind = trace.Kind
		} else {
			kind = v
		}

		if len(a) != 0 {
			format = a[0]
			a = a[1:]
		}
	}

	var err error
	switch value := format.(type) {
	case error:
		err = value

	case string:
		err = fmt.Errorf(value, a...)

	default:
		panic("expected error or string value")
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
