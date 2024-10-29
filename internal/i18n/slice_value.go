package i18n

import "strings"

type Slice []Value

func NewSlice(values []Value) Slice {
	return Slice(values)
}

func (s Slice) Type() Type {
	return TypeSlice
}

func (s Slice) AsBool() Bool {
	return NewBool(len(s) > 0)
}

func (s Slice) AsInt() Int {
	switch n := len(s); n {
	case 0:
		return intZero

	case 1:
		return intOne

	default:
		return NewInt(int64(n))
	}
}

func (s Slice) AsFloat() Float {
	switch n := len(s); n {
	case 0:
		return floatZero

	case 1:
		return floatOne

	default:
		return NewFloat(float64(n))
	}
}

func (s Slice) AsString() String {
	strs := make([]string, len(s))
	for i, value := range s {
		strs[i] = value.AsString().Value
	}

	return NewString(strings.Join(strs, ""))
}

func (s Slice) AsSlice() Slice {
	return s
}

func (s Slice) Equal(rhs Value) Bool {
	if rhs.Type() != TypeSlice {
		return boolFalse
	}

	rhss := rhs.AsSlice()
	if len(s) != len(rhss) {
		return boolFalse
	}

	for i, value := range s {
		if !value.Equal(rhss[i]).Value {
			return boolFalse
		}
	}

	return boolTrue
}

func (s Slice) Less(rhs Value) Bool {
	switch rhs.Type() {
	case TypeInt, TypeFloat:
		return s.AsInt().Less(rhs)

	case TypeSlice:
		return NewBool(len(s) < int(rhs.AsInt().Value))

	default:
		return boolFalse
	}
}

func (s Slice) Add(rhs Value) Value {
	if rhs.Type() == TypeString {
		return s.AsString().Add(rhs)
	}

	return s.AsInt().Add(rhs)
}

func (s Slice) Sub(rhs Value) Value {
	return s.AsInt().Sub(rhs)
}

func (s Slice) Mul(rhs Value) Value {
	return s.AsInt().Mul(rhs)
}

func (s Slice) Div(rhs Value) Value {
	return s.AsInt().Div(rhs)
}

func (s Slice) Mod(rhs Value) Value {
	return s.AsInt().Mod(rhs)
}
