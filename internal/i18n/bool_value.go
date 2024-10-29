package i18n

import "strings"

var (
	boolFalse = Bool{Value: false}
	boolTrue  = Bool{Value: true}
)

type Bool struct {
	Value bool
}

func NewBool(b bool) Bool {
	if b {
		return boolTrue
	}

	return boolFalse
}

func (b Bool) Type() Type {
	return TypeBool
}

func (b Bool) AsBool() Bool {
	return b
}

func (b Bool) AsInt() Int {
	if b.Value {
		return intOne
	}

	return intZero
}

func (b Bool) AsFloat() Float {
	if b.Value {
		return floatOne
	}

	return floatZero
}

func (b Bool) AsString() String {
	if b.Value {
		return stringTrue
	}

	return stringFalse
}

func (b Bool) AsSlice() Slice {
	return NewSlice([]Value{b})
}

func (b Bool) Equal(rhs Value) Bool {
	switch rhs.Type() {
	case TypeBool:
		return NewBool(b.Value == rhs.AsBool().Value)

	case TypeInt:
		return b.AsInt().Equal(rhs)

	case TypeFloat:
		return b.AsFloat().Equal(rhs)
	}

	return boolFalse
}

func (b Bool) Less(rhs Value) Bool {
	switch rhs.Type() {
	case TypeBool, TypeInt:
		return b.AsInt().Less(rhs)

	case TypeFloat:
		return b.AsFloat().Less(rhs)

	default:
		return boolFalse
	}
}

func (b Bool) Add(rhs Value) Value {
	if rhs.Type() == TypeString {
		return b.AsString().Add(rhs)
	}

	return b.AsInt().Add(rhs)
}

func (b Bool) Sub(rhs Value) Value {
	return b.AsInt().Sub(rhs)
}

func (b Bool) Mul(rhs Value) Value {
	return b.AsInt().Mul(rhs)
}

func (b Bool) Div(rhs Value) Value {
	if rhs.Type() == TypeString && strings.Contains(rhs.AsString().Value, ".") {
		return b.AsFloat().Div(rhs)
	}

	return b.AsInt().Div(rhs)
}

func (b Bool) Mod(rhs Value) Value {
	return b.AsInt().Mod(rhs)
}
