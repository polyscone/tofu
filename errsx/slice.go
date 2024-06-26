package errsx

import (
	"errors"
	"strings"
)

// Slice represents a collection of errors.
type Slice []error

// Append pushes the given error onto the end of the slice.
// Any nil errors are ignored and discarded.
func (s *Slice) Append(value any) {
	if value == nil {
		return
	}

	switch value := value.(type) {
	case error:
		if value == nil {
			return
		}

		*s = append(*s, value)

	case []error:
		for _, err := range value {
			if err == nil {
				return
			}

			*s = append(*s, err)
		}

	case string:
		*s = append(*s, errors.New(value))

	case []string:
		for _, str := range value {
			*s = append(*s, errors.New(str))
		}

	default:
		panic("want error or string message")
	}
}

func (s Slice) Err() error {
	if s != nil {
		return s
	}

	return nil
}

func (s Slice) Error() string {
	if s == nil {
		return "<nil>"
	}

	strs := make([]string, len(s))
	for i, err := range s {
		strs[i] = err.Error()
	}

	return strings.Join(strs, "; ")
}

func (s Slice) String() string {
	return s.Error()
}

func (s Slice) As(target any) bool {
	// If the target is an error Map type then we want to
	// collect all of the errors from every map into the target
	if target, ok := target.(*Map); ok {
		var errs Map
		for _, err := range s {
			if errors.As(err, &errs) {
				for key, value := range errs {
					target.Set(key, value)
				}
			}
		}

		if errs != nil {
			return true
		}
	}

	// Fallback for any type that isn't special cased
	for _, err := range s {
		if errors.As(err, target) {
			return true
		}
	}

	return false
}

func (s Slice) Unwrap() []error {
	return s
}
