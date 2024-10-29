package i18n

import "time"

var timeZero = Time{Value: time.Time{}}

type Time struct {
	Value time.Time
}

func NewTime(t time.Time) Time {
	if t.IsZero() {
		return timeZero
	}

	return Time{Value: t}
}

func (t Time) Type() Type {
	return TypeTime
}

func (t Time) AsBool() Bool {
	return NewBool(!t.Value.IsZero())
}

func (t Time) AsInt() Int {
	return NewInt(t.Value.Unix())
}

func (t Time) AsFloat() Float {
	return NewFloat(float64(t.Value.Unix()))
}

func (t Time) AsString() String {
	return NewString(t.Value.Format(time.RFC3339Nano))
}

func (t Time) AsSlice() Slice {
	return NewSlice([]Value{t})
}

func (t Time) Equal(rhs Value) Bool {
	if rhs, ok := rhs.(Time); ok {
		return NewBool(t.Value.Equal(rhs.Value))
	}

	return boolFalse
}

func (t Time) Less(rhs Value) Bool {
	if rhs, ok := rhs.(Time); ok {
		return NewBool(t.Value.Before(rhs.Value))
	}

	return boolFalse
}

func (t Time) Add(rhs Value) Value {
	return timeZero
}

func (t Time) Sub(rhs Value) Value {
	if rhs, ok := rhs.(Time); ok {
		return NewDuration(t.Value.Sub(rhs.Value))
	}

	return durationZero
}

func (t Time) Mul(rhs Value) Value {
	return timeZero
}

func (t Time) Div(rhs Value) Value {
	return timeZero
}

func (t Time) Mod(rhs Value) Value {
	return timeZero
}
