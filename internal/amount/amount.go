package amount

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
)

// RoundingMode indicates which rounding mode should be used when rounding an amount.
type RoundingMode uint8

// Rounding modes to be used with any rounding methods.
const (
	// RoundTruncate will just discard any numbers after the desired places.
	RoundTruncate RoundingMode = iota

	// RoundHalfAwayFromZero will round up when x is positive and down when x is negative.
	RoundHalfAwayFromZero

	// RoundHalfTowardsZero will round down x is positive and up when x is negative.
	RoundHalfTowardsZero

	// RoundHalfToEven (aka "Banker's Rounding") will always round to the closest even number.
	RoundHalfToEven

	// RoundHalfToOdd will always round to the closest odd number number.
	RoundHalfToOdd
)

type Amount struct {
	value  int64
	places int
	unit   string
}

func NewFromInt64(value int64, places int, unit string) Amount {
	return Amount{
		value:  value,
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

			result := amt.value*10 + digit
			if negate && result > amt.value || !negate && result < amt.value {
				return amt, fmt.Errorf("%v overflows 64-bit integer", str)
			}

			amt.value = result

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

	return amt.normalize(), nil
}

func (amt Amount) parts() (string, string) {
	divisor := int64(math.Pow10(amt.places))
	i := strconv.FormatInt(amt.value/divisor, 10)
	if amt.value < 0 && !strings.HasPrefix(i, "-") {
		i = "-" + i
	}

	if amt.places == 0 {
		return i, ""
	}

	f := strconv.FormatInt(int64(math.Abs(float64(amt.value%divisor))), 10)
	if n := len(f); n < amt.places {
		f = strings.Repeat("0", amt.places-n) + f
	}

	return i, f
}

func (amt Amount) Format(minPlaces int) string {
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

func (amt Amount) truncate(places int) Amount {
	for amt.places > places {
		amt.value /= 10
		amt.places--
	}

	return amt
}

func (amt Amount) grow(places int) Amount {
	if places > amt.places {
		n := places - amt.places
		factor := int64(math.Pow10(n))
		result := amt.value * factor

		if factor != 0 && result/factor != amt.value {
			panic(fmt.Sprintf("overflow caused by truncating by a factor of %v on %#v", factor, amt))
		}

		amt.value = result
		amt.places += n
	}

	return amt
}

func (amt Amount) normalize() Amount {
	for amt.places > 0 && amt.value%10 == 0 {
		amt.value /= 10
		amt.places--
	}

	return amt
}

func (amt Amount) Places() int {
	return amt.places
}

func (lhs Amount) Equal(rhs Amount) bool {
	if lhs.unit != "" && rhs.unit != "" && lhs.unit != rhs.unit {
		return false
	}

	lhs = lhs.grow(rhs.places)
	rhs = rhs.grow(lhs.places)

	return lhs.value == rhs.value
}

func (lhs Amount) Add(rhs Amount) Amount {
	if lhs.unit != "" && rhs.unit != "" && lhs.unit != rhs.unit {
		panic(fmt.Sprintf("cannot add two different units: %q and %q", lhs.unit, rhs.unit))
	}

	lhs = lhs.grow(rhs.places)
	rhs = rhs.grow(lhs.places)

	result := lhs.value + rhs.value
	if (result <= lhs.value) == (rhs.value > 0) {
		panic(fmt.Sprintf("overflow caused by addition: %v + %v", lhs, rhs))
	}

	lhs.value = result

	return lhs.normalize()
}

func (lhs Amount) Sub(rhs Amount) Amount {
	if lhs.unit != "" && rhs.unit != "" && lhs.unit != rhs.unit {
		panic(fmt.Sprintf("cannot subtract two different units: %q and %q", lhs.unit, rhs.unit))
	}

	lhs = lhs.grow(rhs.places)
	rhs = rhs.grow(lhs.places)

	result := lhs.value - rhs.value
	if (result >= lhs.value) == (rhs.value > 0) {
		panic(fmt.Sprintf("overflow caused by subtraction: %v - %v", lhs, rhs))
	}

	lhs.value = result

	return lhs.normalize()
}

func (lhs Amount) Mul(rhs Amount) Amount {
	if lhs.unit != "" && rhs.unit != "" && lhs.unit != rhs.unit {
		panic(fmt.Sprintf("cannot multiply two different units: %q and %q", lhs.unit, rhs.unit))
	}

	places := lhs.places + rhs.places
	lhs = lhs.grow(places)
	rhs = rhs.grow(places)

	result := lhs.value * rhs.value
	if rhs.value != 0 && result/rhs.value != lhs.value {
		panic(fmt.Sprintf("overflow caused by multiplication: %v * %v", lhs, rhs))
	}

	divisor := int64(math.Pow10(places))
	lhs.value = result / divisor

	return lhs.normalize()
}

func (amt Amount) Abs() Amount {
	if amt.value < 0 {
		if amt.value == math.MinInt64 {
			panic(fmt.Sprintf("overflow caused by taking the absolute value of the minimum integer"))
		}

		amt.value = -amt.value
	}

	return amt
}

// Round will round the amount to the given places using the given rounding mode.
// If the places given is larger than the current amount's places then it's ignored.
// If the places is negative then it will be treated as 0.
func (amt Amount) Round(places int, mode RoundingMode) Amount {
	// If the places we want to round to is the same as the current places then we have nothing to do
	if places < 0 {
		places = 0
	}
	if places >= amt.places {
		return amt
	}

	// If the mode is set to truncate we can just truncate without doing anything else
	if mode == RoundTruncate {
		return amt.truncate(places)
	}

	// For the rest of the modes we need to know what the last digit is one place ahead of the places we want to round
	// to, which is why we add 1 to the places, and we need an Amount value set to 5 with the same number of places
	// which we can use to add or subtract before truncation
	last := amt.truncate(places+1).value % 10
	adjust := NewFromInt64(5, places+1, "")

	switch mode {
	case RoundHalfAwayFromZero:
		// If the last integer is +5 or more, then we need to add 5 before truncating
		// The truncate will truncate the final digit leaving us with the nearest result away from zero
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

	case RoundHalfTowardsZero:
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

	case RoundHalfToEven:
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
		roundedIsEven := amt.truncate(places).value%2 == 0
		if (last == 5 || last == -5) && roundedIsEven {
			return amt.truncate(places)
		}

		if last >= 0 {
			return amt.Add(adjust).truncate(places)
		}

		return amt.Sub(adjust).truncate(places)

	case RoundHalfToOdd:
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
		roundedIsOdd := amt.truncate(places).value%2 != 0
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
	amt.value = 0
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
