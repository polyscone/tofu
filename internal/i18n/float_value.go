package i18n

import (
	"strconv"
)

var (
	floatZero = Float{Value: 0}
	floatOne  = Float{Value: 1}
)

type Float struct {
	Value float64
}

func NewFloat(f float64) Float {
	switch f {
	case 0:
		return floatZero

	case 1:
		return floatOne

	default:
		return Float{Value: f}
	}
}

func (f Float) Type() Type {
	return TypeFloat
}

func (f Float) AsBool() Bool {
	return NewBool(f.Value != 0)
}

func (f Float) AsInt() Int {
	return NewInt(int64(f.Value))
}

func (f Float) AsFloat() Float {
	return f
}

func (f Float) AsString() String {
	str := strconv.FormatFloat(f.Value, 'g', -1, 64)

	return NewString(str)
}

func (f Float) AsSlice() Slice {
	return NewSlice([]Value{f})
}

func (f Float) Equal(rhs Value) Bool {
	return NewBool(f.Value == rhs.AsFloat().Value)
}

func (f Float) Less(rhs Value) Bool {
	if rhs.Type() == TypeString {
		return f.AsString().Less(rhs)
	}

	return NewBool(f.Value < rhs.AsFloat().Value)
}

func (f Float) Add(rhs Value) Value {
	if rhs.Type() == TypeString {
		return f.AsString().Add(rhs)
	}

	return NewFloat(f.Value + rhs.AsFloat().Value)
}

func (f Float) Sub(rhs Value) Value {
	return NewFloat(f.Value - rhs.AsFloat().Value)
}

func (f Float) Mul(rhs Value) Value {
	if rhs.Type() == TypeString {
		return f.AsInt().Mul(rhs)
	}

	return NewFloat(f.Value * rhs.AsFloat().Value)
}

func (f Float) Div(rhs Value) Value {
	rhsf := rhs.AsFloat().Value
	if rhsf == 0 {
		return floatZero
	}

	return NewFloat(f.Value / rhsf)
}

func (f Float) Mod(rhs Value) Value {
	return f.AsInt().Mod(rhs)
}
