package i18n

import "time"

var durationZero = Duration{Value: time.Duration(0)}

type Duration struct {
	Value time.Duration
}

func NewDuration(d time.Duration) Duration {
	if d == 0 {
		return durationZero
	}

	return Duration{Value: d}
}

func (d Duration) Type() Type {
	return TypeDuration
}

func (d Duration) AsBool() Bool {
	return NewBool(d.Value != 0)
}

func (d Duration) AsInt() Int {
	return NewInt(int64(d.Value))
}

func (d Duration) AsFloat() Float {
	return NewFloat(float64(d.Value))
}

func (d Duration) AsString() String {
	return NewString(d.Value.String())
}

func (d Duration) AsSlice() Slice {
	return NewSlice([]Value{d})
}

func (d Duration) Equal(rhs Value) Bool {
	return d.AsInt().Equal(rhs)
}

func (d Duration) Less(rhs Value) Bool {
	return d.AsInt().Less(rhs)
}

func (d Duration) Add(rhs Value) Value {
	return NewDuration(time.Duration(d.AsInt().Add(rhs).AsInt().Value))
}

func (d Duration) Sub(rhs Value) Value {
	return NewDuration(time.Duration(d.AsInt().Sub(rhs).AsInt().Value))
}

func (d Duration) Mul(rhs Value) Value {
	return NewDuration(time.Duration(d.AsInt().Mul(rhs).AsInt().Value))
}

func (d Duration) Div(rhs Value) Value {
	return NewDuration(time.Duration(d.AsInt().Div(rhs).AsInt().Value))
}

func (d Duration) Mod(rhs Value) Value {
	return NewDuration(time.Duration(d.AsInt().Mod(rhs).AsInt().Value))
}
