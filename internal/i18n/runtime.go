package i18n

import (
	"errors"
	"fmt"
	"time"
)

const (
	TypeUnknown Type = iota
	TypeBool
	TypeInt
	TypeFloat
	TypeString
	TypeRawString
	TypeTime
	TypeDuration
	TypeSlice
)

var funcNames = map[string]Value{
	"join":       NewString("join"),
	"split":      NewString("split"),
	"bold":       NewString("bold"),
	"italic":     NewString("italic"),
	"link":       NewString("link"),
	"pad_left":   NewString("pad_left"),
	"pad_right":  NewString("pad_right"),
	"trim_left":  NewString("trim_left"),
	"trim_right": NewString("trim_right"),
	"integer":    NewString("integer"),
	"fraction":   NewString("fraction"),
	"t":          NewString("t"),
}

var typeNames = [...]string{
	TypeUnknown:   "unknown",
	TypeBool:      "bool",
	TypeInt:       "int",
	TypeFloat:     "float",
	TypeString:    "string",
	TypeRawString: "rawstring",
	TypeTime:      "time",
	TypeDuration:  "duration",
	TypeSlice:     "slice",
}

var (
	DefaultHTMLRuntime     HTMLRuntime
	DefaultJSRuntime       JSRuntime
	DefaultMarkdownRuntime MarkdownRuntime
)

type AfterPostProcessFunc func(res string) string

type Runtime interface {
	Kind() string
	Join(s, sep Value) String
	Split(s, sep Value) Slice
	Bold(value Value) RawString
	Italic(value Value) RawString
	Link(label, href, target Value) RawString
	PadLeft(value, length, padding Value) String
	PadRight(value, length, padding Value) String
	TrimLeft(value, trim Value) String
	TrimRight(value, trim Value) String
	Integer(value Value) Int
	Fraction(value, roundingUnit Value) Float
	T(key Value, locale string, value, opt Value) String
	PostProcess(value Value, after AfterPostProcessFunc) any
}

type Type int

func (t Type) String() string {
	return typeNames[t]
}

type Value interface {
	Type() Type
	AsBool() Bool
	AsInt() Int
	AsFloat() Float
	AsString() String
	AsSlice() Slice
	Equal(rhs Value) Bool
	Less(rhs Value) Bool
	Add(rhs Value) Value
	Sub(rhs Value) Value
	Mul(rhs Value) Value
	Div(rhs Value) Value
	Mod(rhs Value) Value
}

func NewValue(v any) (Value, error) {
	switch v := v.(type) {
	case Value:
		return v, nil

	case bool:
		return NewBool(v), nil

	case int:
		return NewInt(int64(v)), nil

	case int8:
		return NewInt(int64(v)), nil

	case int16:
		return NewInt(int64(v)), nil

	case int32:
		return NewInt(int64(v)), nil

	case int64:
		return NewInt(v), nil

	case uint:
		return NewInt(int64(v)), nil

	case uint8:
		return NewInt(int64(v)), nil

	case uint16:
		return NewInt(int64(v)), nil

	case uint32:
		return NewInt(int64(v)), nil

	case uint64:
		return NewInt(int64(v)), nil

	case float32:
		return NewFloat(float64(v)), nil

	case float64:
		return NewFloat(v), nil

	case []byte:
		return NewString(string(v)), nil

	case string:
		return NewString(v), nil

	case time.Time:
		return NewTime(v), nil

	case time.Duration:
		return NewDuration(v), nil

	case []bool:
		return NewValues(v)

	case []int:
		return NewValues(v)

	case []int8:
		return NewValues(v)

	case []int16:
		return NewValues(v)

	case []int32:
		return NewValues(v)

	case []int64:
		return NewValues(v)

	case []float32:
		return NewValues(v)

	case []float64:
		return NewValues(v)

	case [][]byte:
		return NewValues(v)

	case []string:
		return NewValues(v)

	case []any:
		return NewValues(v)

	default:
		return nil, fmt.Errorf("could not convert %T to i18n.Value", v)
	}
}

func NewValues[T any](s []T) (Slice, error) {
	values := make(Slice, len(s))
	for i, el := range s {
		value, err := NewValue(el)
		if err != nil {
			return nil, err
		}

		values[i] = value
	}

	return values, nil
}

type Vars map[string]Value

func NewVars(pairs []any) (Vars, error) {
	if len(pairs) == 0 {
		return nil, nil
	}

	if len(pairs)%2 == 1 {
		return nil, errors.New("want key value pairs")
	}

	vars := make(Vars, len(pairs)/2)
	for i := 0; i < len(pairs); i += 2 {
		key := fmt.Sprintf("%v", pairs[i])
		value, err := NewValue(pairs[i+1])
		if err != nil {
			return nil, fmt.Errorf("translation key %q: %w", key, err)
		}

		vars[key] = value
	}

	return vars, nil
}
