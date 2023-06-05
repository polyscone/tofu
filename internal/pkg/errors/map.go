package errors

import (
	"fmt"
	"strings"
)

// Map represents a collection of errors keyed by name.
type Map map[string]error

// Tracef returns nil if the map is nil.
// If the map is not nil then a traced error is created using the given error or
// format string.
func (m Map) Tracef(format any, a ...any) error {
	if m == nil {
		return nil
	}

	trace := tracef(2, format, a...).(Trace)
	trace.fields = m

	return trace
}

// Get will return the error string for the given key.
func (m Map) Get(key string) string {
	if err := m[key]; err != nil {
		return err.Error()
	}

	return ""
}

// Set associates the given error with the given key.
// The map is lazily instantiated if it is nil.
func (m *Map) Set(key string, err any) {
	if *m == nil {
		*m = make(Map)
	}

	(*m)[key] = tracef(2, err)
}

func (m Map) String() string {
	if m == nil {
		return "<nil>"
	}

	pairs := make([]string, len(m))
	i := 0
	for key, err := range m {
		pairs[i] = fmt.Sprintf("%v: %v", key, err)

		i++
	}

	return strings.Join(pairs, "; ")
}

// MarshalJSON implements the json.Marshaler interface.
func (m Map) MarshalJSON() ([]byte, error) {
	errs := make([]string, 0, len(m))
	for key, err := range m {
		errs = append(errs, fmt.Sprintf("%q:%q", key, err.Error()))
	}

	return []byte(fmt.Sprintf("{%v}", strings.Join(errs, ", "))), nil
}
