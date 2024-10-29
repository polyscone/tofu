package i18n

import "strconv"

var (
	intZero = Int{Value: 0}
	intOne  = Int{Value: 1}
)

type Int struct {
	Value int64
}

func NewInt(i int64) Int {
	switch i {
	case 0:
		return intZero

	case 1:
		return intOne

	default:
		return Int{Value: i}
	}
}

func (i Int) Type() Type {
	return TypeInt
}

func (i Int) AsBool() Bool {
	return NewBool(i.Value != 0)
}

func (i Int) AsInt() Int {
	return i
}

func (i Int) AsFloat() Float {
	return NewFloat(float64(i.Value))
}

func (i Int) AsString() String {
	switch i.Value {
	case 0:
		return stringZero

	case 1:
		return stringOne

	default:
		str := strconv.FormatInt(i.Value, 10)

		return NewString(str)
	}
}

func (i Int) AsSlice() Slice {
	return NewSlice([]Value{i})
}

func (i Int) Equal(rhs Value) Bool {
	if rhs.Type() == TypeFloat {
		return i.AsFloat().Equal(rhs)
	}

	return NewBool(i.Value == rhs.AsInt().Value)
}

func (i Int) Less(rhs Value) Bool {
	switch rhs.Type() {
	case TypeBool, TypeInt:
		return NewBool(i.Value < rhs.AsInt().Value)

	case TypeFloat:
		return i.AsFloat().Less(rhs)

	case TypeString:
		return i.AsString().Less(rhs)

	default:
		return boolFalse
	}
}

func (i Int) Add(rhs Value) Value {
	switch rhs.Type() {
	case TypeFloat:
		return i.AsFloat().Add(rhs)

	case TypeString:
		return i.AsString().Add(rhs)
	}

	return NewInt(i.Value + rhs.AsInt().Value)
}

func (i Int) Sub(rhs Value) Value {
	if rhs.Type() == TypeFloat {
		return i.AsFloat().Sub(rhs)
	}

	return NewInt(i.Value - rhs.AsInt().Value)
}

func (i Int) Mul(rhs Value) Value {
	switch rhs.Type() {
	case TypeFloat:
		return i.AsFloat().Mul(rhs)

	case TypeString:
		return rhs.Mul(i)
	}

	return NewInt(i.Value * rhs.AsInt().Value)
}

func (i Int) Div(rhs Value) Value {
	if rhs.Type() == TypeFloat {
		return i.AsFloat().Div(rhs)
	}

	rhsi := rhs.AsInt().Value
	if rhsi == 0 {
		return intZero
	}

	return NewInt(i.Value / rhsi)
}

func (i Int) Mod(rhs Value) Value {
	rhsi := rhs.AsInt().Value
	if rhsi == 0 {
		return intZero
	}

	return NewInt(i.Value % rhsi)
}
