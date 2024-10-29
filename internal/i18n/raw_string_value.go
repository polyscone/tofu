package i18n

import (
	"strconv"
	"strings"
)

var (
	rawStringEmpty = RawString{Value: ""}
	rawStringFalse = RawString{Value: "false"}
	rawStringTrue  = RawString{Value: "true"}
	rawStringZero  = RawString{Value: "0"}
	rawStringOne   = RawString{Value: "1"}
)

type RawString struct {
	Value string
}

func NewRawString(s string) RawString {
	switch s {
	case "":
		return rawStringEmpty

	case "false":
		return rawStringFalse

	case "true":
		return rawStringTrue

	case "0":
		return rawStringZero

	case "1":
		return rawStringOne

	default:
		return RawString{Value: s}
	}
}

func (s RawString) Type() Type {
	return TypeRawString
}

func (s RawString) AsBool() Bool {
	return NewBool(s.Value != "")
}

func (s RawString) AsInt() Int {
	switch s.Value {
	case "0", "":
		return intZero

	case "1":
		return intOne

	default:
		i, err := strconv.ParseInt(s.Value, 10, 64)
		if err != nil {
			return intZero
		}

		return NewInt(i)
	}
}

func (s RawString) AsFloat() Float {
	switch s.Value {
	case "0", "0.0", "0.00", "0.000", "0.0000", "0.00000", "":
		return floatZero

	case "1", "1.0", "1.00", "1.000", "1.0000", "1.00000":
		return floatOne

	default:
		f, err := strconv.ParseFloat(s.Value, 64)
		if err != nil {
			return floatZero
		}

		return NewFloat(f)
	}
}

func (s RawString) AsString() String {
	return NewString(s.Value)
}

func (s RawString) AsSlice() Slice {
	runes := []rune(s.Value)
	values := make([]Value, len(runes))
	for i, r := range runes {
		values[i] = NewString(string(r))
	}

	return NewSlice(values)
}

func (s RawString) Equal(rhs Value) Bool {
	return NewBool(s.Value == rhs.AsString().Value)
}

func (s RawString) Less(rhs Value) Bool {
	switch rhs.Type() {
	case TypeInt, TypeFloat, TypeString:
		return NewBool(s.Value < rhs.AsString().Value)

	default:
		return boolFalse
	}
}

func (s RawString) Add(rhs Value) Value {
	return NewString(s.Value + rhs.AsString().Value)
}

func (s RawString) Sub(rhs Value) Value {
	return s.AsInt().Sub(rhs)
}

func (s RawString) Mul(rhs Value) Value {
	switch rhs.Type() {
	case TypeBool, TypeInt, TypeFloat, TypeSlice:
		return NewString(strings.Repeat(s.Value, int(rhs.AsInt().Value)))
	}

	i, err := strconv.ParseInt(s.Value, 10, 64)
	if err == nil {
		return NewString(strings.Repeat(rhs.AsString().Value, int(i)))
	}

	return NewString(strings.Repeat(s.Value, int(rhs.AsInt().Value)))
}

func (s RawString) Div(rhs Value) Value {
	return s.AsInt().Div(rhs)
}

func (s RawString) Mod(rhs Value) Value {
	return s.AsInt().Mod(rhs)
}
