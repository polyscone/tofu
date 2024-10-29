package i18n

import (
	"strconv"
	"strings"
)

var (
	stringEmpty = String{Value: ""}
	stringFalse = String{Value: "false"}
	stringTrue  = String{Value: "true"}
	stringZero  = String{Value: "0"}
	stringOne   = String{Value: "1"}
	stringError = String{Value: "{i18n: error}"}
)

type String struct {
	Value string
}

func NewString(s string) String {
	switch s {
	case "":
		return stringEmpty

	case "false":
		return stringFalse

	case "true":
		return stringTrue

	case "0":
		return stringZero

	case "1":
		return stringOne

	default:
		return String{Value: s}
	}
}

func (s String) Type() Type {
	return TypeString
}

func (s String) AsBool() Bool {
	return NewBool(s.Value != "")
}

func (s String) AsInt() Int {
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

func (s String) AsFloat() Float {
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

func (s String) AsString() String {
	return s
}

func (s String) AsSlice() Slice {
	runes := []rune(s.Value)
	values := make([]Value, len(runes))
	for i, r := range runes {
		values[i] = NewString(string(r))
	}

	return NewSlice(values)
}

func (s String) Equal(rhs Value) Bool {
	return NewBool(s.Value == rhs.AsString().Value)
}

func (s String) Less(rhs Value) Bool {
	switch rhs.Type() {
	case TypeInt, TypeFloat, TypeString:
		return NewBool(s.Value < rhs.AsString().Value)

	default:
		return boolFalse
	}
}

func (s String) Add(rhs Value) Value {
	return NewString(s.Value + rhs.AsString().Value)
}

func (s String) Sub(rhs Value) Value {
	return s.AsInt().Sub(rhs)
}

func (s String) Mul(rhs Value) Value {
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

func (s String) Div(rhs Value) Value {
	return s.AsInt().Div(rhs)
}

func (s String) Mod(rhs Value) Value {
	return s.AsInt().Mod(rhs)
}
