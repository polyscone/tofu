package amount

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/big"
	"strings"
)

var (
	int0  = big.NewInt(0)
	int1  = big.NewInt(1)
	int10 = big.NewInt(10)
)

var Zero = NewFromInt64(0, 0, "")

// RoundingMode indicates which rounding mode should be used when rounding an amount.
type RoundingMode uint8

// Rounding modes to be used with any rounding methods.
const (
	// Truncate will just discard any numbers after the desired places.
	Truncate RoundingMode = iota

	// HalfAwayFromZero will round up when x is positive and down when x is negative.
	HalfAwayFromZero

	// HalfTowardsZero will round down x is positive and up when x is negative.
	HalfTowardsZero

	// HalfToEven (aka "Banker's Rounding") will always round to the closest even number.
	HalfToEven

	// HalfToOdd will always round to the closest odd number number.
	HalfToOdd
)

type Amount struct {
	value  *big.Int
	places int
	unit   string
}

func NewFromInt64(value int64, places int, unit string) Amount {
	return Amount{
		value:  big.NewInt(value),
		places: places,
		unit:   unit,
	}
}

func New(str string) (Amount, error) {
	amt := NewFromInt64(0, 0, "")

	var negate bool
	mode := "int"
	for _, r := range str {
		switch {
		case r == ' ':
			mode = "curr"

		case r == ',':
			continue

		case r == '-':
			if mode != "int" {
				return amt, errors.New("negation must come first")
			}

			negate = !negate

		case r == '+':
			if mode != "int" {
				return amt, errors.New("negation must come first")
			}

			negate = false

		case r >= '0' && r <= '9':
			digit := int64(r - '0')
			if negate {
				digit = -digit
			}

			ones := big.NewInt(digit)

			amt.value.Mul(amt.value, int10).Add(amt.value, ones)

			if mode == "frac" {
				amt.places++
			}

		case r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z':
			if mode != "curr" {
				return amt, errors.New("unit must come last")
			}

			amt.unit += string(r)

		case r == '.':
			switch mode {
			case "int":
				mode = "frac"

			case "frac":
				return amt, errors.New("must only contain one decimal point")

			case "curr":
				return amt, errors.New("decimal point must come before unit")
			}

		default:
			return amt, fmt.Errorf("unexpected %q", r)
		}
	}

	return amt, nil
}

func (amt Amount) parts() (string, string) {
	if amt.value == nil {
		return "0", ""
	}

	d := fmt.Sprintf("%+d", amt.value)

	d = strings.TrimPrefix(d, "+")

	if len(d) < amt.places {
		d = strings.Repeat("0", amt.places-len(d)) + d
	}

	p := len(d) - amt.places
	i := d[:p]
	f := d[p:]

	switch i {
	case "":
		i = "0"

	case "-":
		i = "-0"
	}

	return i, f
}

func (amt Amount) Format(minPlaces int) string {
	if minPlaces > 0 {
		amt = amt.normalize()
	}

	places := max(minPlaces, amt.places)
	i, f := amt.parts()
	if places == 0 {
		return i
	}

	if n := places - len(f); n > 0 {
		f += strings.Repeat("0", n)
	}

	if f != "" {
		return i + "." + f
	}

	return i
}

func (amt Amount) String() string {
	if amt.unit != "" {
		return amt.Format(0) + " " + amt.unit
	}

	return amt.Format(0)
}

func (amt Amount) IsZero() bool {
	return amt.Equal(Zero)
}

func (amt Amount) Places() int {
	return amt.places
}

func (amt Amount) Unit() string {
	return amt.unit
}

func (amt Amount) Int64() (int64, int, bool) {
	if amt.value == nil {
		return 0, amt.Places(), true
	}

	if !amt.value.IsInt64() {
		return 0, 0, false
	}

	return amt.value.Int64(), amt.Places(), true
}

func (amt Amount) Int() (int, int, bool) {
	i64, places, ok := amt.Int64()
	if !ok {
		return 0, 0, false
	}

	i := int(i64)
	if int64(i) != i64 {
		return 0, 0, false
	}

	return i, places, true
}

func (amt Amount) copy() Amount {
	if amt.value == nil {
		amt.value = big.NewInt(0)
	}

	return Amount{
		value:  big.NewInt(0).Set(amt.value),
		places: amt.places,
		unit:   amt.unit,
	}
}

func (amt Amount) truncate(places int) Amount {
	amt = amt.copy()
	for amt.places > places {
		amt.value.Quo(amt.value, int10)
		amt.places--
	}

	return amt
}

func (amt Amount) grow(places int) Amount {
	amt = amt.copy()
	if places > amt.places {
		n := places - amt.places
		factor := big.NewInt(int64(math.Pow10(n)))

		amt.value.Mul(amt.value, factor)

		amt.places += n
	}

	return amt
}

func (amt Amount) normalize() Amount {
	amt = amt.copy()
	for amt.places > 0 {
		x := big.NewInt(0).Set(amt.value)
		if x.Rem(x, int10).Cmp(int0) != 0 {
			break
		}

		amt.value.Quo(amt.value, int10)
		amt.places--
	}

	return amt
}

func (lhs Amount) Equal(rhs Amount) bool {
	if lhs.unit != "" && rhs.unit != "" && lhs.unit != rhs.unit {
		return false
	}

	lhs = lhs.grow(rhs.places)
	rhs = rhs.grow(lhs.places)

	return lhs.value.Cmp(rhs.value) == 0
}

func (lhs Amount) Less(rhs Amount) bool {
	if lhs.unit != "" && rhs.unit != "" && lhs.unit != rhs.unit {
		return false
	}

	lhs = lhs.grow(rhs.places)
	rhs = rhs.grow(lhs.places)

	return lhs.value.Cmp(rhs.value) < 0
}

func (lhs Amount) LessEqual(rhs Amount) bool {
	return lhs.Less(rhs) || lhs.Equal(rhs)
}

func (lhs Amount) Greater(rhs Amount) bool {
	if lhs.unit != "" && rhs.unit != "" && lhs.unit != rhs.unit {
		return false
	}

	lhs = lhs.grow(rhs.places)
	rhs = rhs.grow(lhs.places)

	return lhs.value.Cmp(rhs.value) > 0
}

func (lhs Amount) GreaterEqual(rhs Amount) bool {
	return lhs.Greater(rhs) || lhs.Equal(rhs)
}

func (amt Amount) WithMinPlaces(places int) Amount {
	if places < 0 {
		places = 0
	}

	amt = amt.grow(places)

	return amt
}

func (lhs Amount) Add(rhs Amount) Amount {
	if lhs.unit != "" && rhs.unit != "" && lhs.unit != rhs.unit {
		panic(fmt.Sprintf("cannot add two different units: %q and %q", lhs.unit, rhs.unit))
	}

	lhs = lhs.grow(rhs.places)
	rhs = rhs.grow(lhs.places)

	lhs.value.Add(lhs.value, rhs.value)

	return lhs
}

func (lhs Amount) Sub(rhs Amount) Amount {
	if lhs.unit != "" && rhs.unit != "" && lhs.unit != rhs.unit {
		panic(fmt.Sprintf("cannot subtract two different units: %q and %q", lhs.unit, rhs.unit))
	}

	lhs = lhs.grow(rhs.places)
	rhs = rhs.grow(lhs.places)

	lhs.value.Sub(lhs.value, rhs.value)

	return lhs
}

func (lhs Amount) Mul(rhs Amount) Amount {
	if lhs.unit != "" && rhs.unit != "" && lhs.unit != rhs.unit {
		panic(fmt.Sprintf("cannot multiply two different units: %q and %q", lhs.unit, rhs.unit))
	}

	places := lhs.places + rhs.places
	lhs = lhs.grow(places)
	rhs = rhs.grow(places)

	divisor := big.NewInt(int64(math.Pow10(places)))
	lhs.value.Mul(lhs.value, rhs.value)
	lhs.value.Quo(lhs.value, divisor)

	return lhs
}

// AllocateBetween will split the given amount into the number of portions provided.
//
// It returns a slice of amounts which sum to the original amount and a split index.
// The split index points to the amount in the returned slice that allocation stopped
// at, and describes the index in the slice where the amounts change in value.
//
// For example, if 10.00 is allocated between 3 portions the returned slice will
// be {3.34, 3.33, 3.33} and the split index will be 1. This means that index 0 was the
// last index to receive allocation and the split index of 1 shows where the amounts
// change from 3.34 to 3.33. A split index of 0 means that all amounts in the returned
// slice have the same value.
func (amt Amount) AllocateBetween(portions int) ([]Amount, int) {
	if portions <= 0 {
		return nil, 0
	}

	amt = amt.copy()

	var i int
	amounts := make([]Amount, portions)
	for i := range len(amounts) {
		amounts[i] = NewFromInt64(0, amt.Places(), "")
	}
	for amt.value.Cmp(int0) != 0 {
		amounts[i%portions].value.Add(amounts[i%portions].value, int1)

		amt.value.Sub(amt.value, int1)

		i++
	}

	split := i % portions

	return amounts, split
}

func (amt Amount) Abs() Amount {
	amt = amt.copy()

	amt.value.Abs(amt.value)

	return amt
}

func (amt Amount) Neg() Amount {
	amt = amt.copy()

	amt.value.Neg(amt.value)

	return amt
}

// Round will round the amount to the given places using the given rounding mode.
// If the places given is larger than the current amount's places then it's ignored.
// If the places is negative then it will be treated as 0.
func (amt Amount) Round(places int, mode RoundingMode) Amount {
	amt = amt.copy()

	// If the places we want to round to is the same as the current places then we have nothing to do
	if places < 0 {
		places = 0
	}
	if places >= amt.places {
		return amt
	}

	// If the mode is set to truncate we can just truncate without doing anything else
	if mode == Truncate {
		return amt.truncate(places)
	}

	// For the rest of the modes we need to know what the last digit is one place ahead of the places we want to round
	// to, which is why we add 1 to the places, and we need an Amount value set to 5 with the same number of places
	// which we can use to add or subtract before truncation
	trunc := amt.truncate(places + 1)
	last := big.NewInt(0).Set(trunc.value).Rem(trunc.value, int10).Int64()
	adjust := NewFromInt64(5, places+1, "")

	switch mode {
	case HalfAwayFromZero:
		// If the last integer is +5 or more, then we need to add 5 before truncating
		// The truncate will remove the final digit leaving us with the nearest result away from zero
		// Otherwise we always subtract 5 before truncating
		// For example, rounding to a places of 1:
		//    0.06 ->  0.11 ->  0.1
		//    0.05 ->  0.10 ->  0.1
		//    0.04 -> -0.01 ->  0.0
		//    0.00 -> -0.05 ->  0.0
		//   -0.06 -> -0.11 -> -0.1
		//   -0.05 -> -0.10 -> -0.1
		//   -0.04 -> -0.09 ->  0.0
		if last >= 5 {
			return amt.Add(adjust).truncate(places)
		} else if last <= -5 {
			return amt.Sub(adjust).truncate(places)
		}

		return amt.truncate(places)

	case HalfTowardsZero:
		// If the number is positive then we subtract 5 before truncating if the last digits is less than or equal to 5,
		// otherwise we add 5 before truncating
		// If the number is negative then we add 5 before truncating if the last digits is greater than or equal to -5,
		// otherwise we subtract 5 before truncating
		// In the case where the last digit is a 0 we can just truncate immediately
		// For example, rounding to a places of 1:
		//    0.06 ->  0.11 ->  0.1
		//    0.05 ->  0.00 ->  0.0
		//    0.04 -> -0.01 ->  0.0
		//    0.00 ->       ->  0.0
		//   -0.06 -> -0.11 -> -0.1
		//   -0.05 ->  0.00 ->  0.0
		//   -0.04 ->  0.01 ->  0.0
		if last > 5 {
			return amt.Add(adjust).truncate(places)
		} else if last < -5 {
			return amt.Sub(adjust).truncate(places)
		}

		return amt.truncate(places)

	case HalfToEven:
		// If the last digit is +5 or -5 and the digit before the last one is even then we can just truncate, because
		// in that case the number that comes before the last digit will always be the closest even number
		// Otherwise, in the case of a positive number, we can add 5 and then truncate
		// Or in the case of a negative number we can subtract 5 and then truncate
		// For example, rounding to a places of 1:
		//    0.06 ->  0.11 ->  0.1
		//    0.05 ->       ->  0.0
		//    0.15 ->  0.20 ->  0.2
		//    0.04 ->  0.09 ->  0.0
		//    0.00 ->  0.05 ->  0.0
		//   -0.06 -> -0.11 -> -0.1
		//   -0.05 ->       ->  0.0
		//   -0.15 -> -0.20 -> -0.2
		//   -0.04 -> -0.09 ->  0.0
		roundedIsEven := amt.copy().truncate(places).value.Bit(0) == 0
		if (last == 5 || last == -5) && roundedIsEven {
			return amt.truncate(places)
		}

		if last >= 0 {
			return amt.Add(adjust).truncate(places)
		}

		return amt.Sub(adjust).truncate(places)

	case HalfToOdd:
		// If the last digit is +5 or -5 and the digit before the last one is odd then we can just truncate, because
		// in that case the number that comes before the last digit will always be the closest odd number
		// Otherwise, in the case of a positive number, we can add 5 and then truncate
		// Or in the case of a negative number we can subtract 5 and then truncate
		// For example, rounding to a places of 1:
		//    0.06 ->  0.11 ->  0.1
		//    0.05 ->  0.10 ->  0.1
		//    0.15 ->       ->  0.1
		//    0.04 ->  0.09 ->  0.0
		//    0.00 ->  0.05 ->  0.0
		//   -0.06 -> -0.11 -> -0.1
		//   -0.05 -> -0.10 -> -0.1
		//   -0.15 ->       -> -0.1
		//   -0.04 -> -0.09 ->  0.0
		roundedIsOdd := amt.copy().truncate(places).value.Bit(0) == 1
		if (last == 5 || last == -5) && roundedIsOdd {
			return amt.truncate(places)
		}

		if last >= 0 {
			return amt.Add(adjust).truncate(places)
		}

		return amt.Sub(adjust).truncate(places)

	default:
		panic("unsupported rounding mode")
	}
}

// Scan implements the SQL Scanner interface for Amount.
func (amt *Amount) Scan(src any) error {
	amt.places = 0
	amt.unit = ""

	var val string
	switch v := src.(type) {
	case []byte:
		val = string(v)

	case string:
		val = v

	default:
		return fmt.Errorf("amount: sql: type %T not supported", v)
	}

	_amt, err := New(val)
	if err != nil {
		return err
	}

	amt.value = _amt.value
	amt.places = _amt.places
	amt.unit = _amt.unit

	return nil
}

// Value implements the SQL Valuer interface for Amount.
func (amt Amount) Value() (driver.Value, error) {
	return amt.String(), nil
}

// MarshalJSON implements the JSON Marshaler interface for Amount.
func (amt Amount) MarshalJSON() ([]byte, error) {
	return json.Marshal(amt.String())
}

// UnmarshalJSON implements the JSON Unmarshaler interface for Amount.
func (amt *Amount) UnmarshalJSON(data []byte) error {
	str := strings.Trim(string(data), `"`)
	_amt, err := New(str)
	if err != nil {
		return err
	}

	amt.value = _amt.value
	amt.places = _amt.places
	amt.unit = _amt.unit

	return nil
}
