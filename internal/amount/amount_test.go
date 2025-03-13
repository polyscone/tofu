package amount_test

import (
	"encoding/json"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/polyscone/tofu/internal/amount"
	"github.com/polyscone/tofu/internal/errsx"
)

func TestNew(t *testing.T) {
	tt := []struct {
		name  string
		value string
	}{
		{"min int64", "-9223372036854775808"},
		{"max int64", "9223372036854775807"},
		{"min int64 overflow", "-109223372036854775808"},
		{"max int64 overflow", "109223372036854775807"},
		{"negative", "-123"},
		{"negative with unit", "-123 kg"},
		{"positive", "+123"},
		{"positive with unit", "+123 kg"},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			errsx.Must(amount.New(tc.value))
		})
	}
}

func TestArithmetic(t *testing.T) {
	tt := []struct {
		a    string
		op   string
		b    string
		want string
	}{
		{"0.1", "+", "0.2", "0.3"},
		{"-0.1", "+", "0.2", "0.1"},
		{"0.1", "+", "-0.2", "-0.1"},
		{"-0.1", "+", "-0.2", "-0.3"},
		{"100", "+", "2.123", "102.123"},
		{"100.000", "+", "2", "102"},
		{"100.020", "+", "0", "100.02"},
		{"100.000", "+", "200.123", "300.123"},
		{"0005.000", "+", "0000002.000000", "7"},
		{"10000000000000000000.000000001", "+", "10000000000000000000.000000001", "20000000000000000000.000000002"},

		{"0.1", "-", "0.2", "-0.1"},
		{"-0.1", "-", "0.2", "-0.3"},
		{"0.1", "-", "-0.2", "0.3"},
		{"-0.1", "-", "-0.2", "0.1"},
		{"100", "-", "2.123", "97.877"},
		{"100.000", "-", "2", "98"},
		{"100.020", "-", "0", "100.02"},
		{"100.000", "-", "200.123", "-100.123"},
		{"0005.000", "-", "0000002.000000", "3"},
		{"10000000000000000000.000000001", "-", "10000000000000000000.000000001", "0"},

		{"0.1", "*", "0.2", "0.02"},
		{"-0.1", "*", "0.2", "-0.02"},
		{"0.1", "*", "-0.2", "-0.02"},
		{"-0.1", "*", "-0.2", "0.02"},
		{"100", "*", "2.123", "212.3"},
		{"100.000", "*", "2", "200"},
		{"100.020", "*", "0", "0"},
		{"100.000", "*", "200.123", "20012.3"},
		{"0005.000", "*", "0000002.000000", "10"},
		{"1.10", "*", "0.5227", "0.57497"},
		{"10000000000000000000.000000001", "*", "10000000000000000000.000000001", "100000000000000000000000000020000000000.000000000000000001"},

		{"1.10", "abs", "", "1.10"},
		{"-1.10", "abs", "", "1.10"},
		{"-10000000000000000000.000000001", "abs", "", "10000000000000000000.000000001"},
	}
	for i, tc := range tt {
		t.Run("test["+strconv.Itoa(i)+"]", func(t *testing.T) {
			a := errsx.Must(amount.New(tc.a))
			b := errsx.Must(amount.New(tc.b))
			want := errsx.Must(amount.New(tc.want))

			var got amount.Amount
			switch tc.op {
			case "+":
				got = a.Add(b)

			case "-":
				got = a.Sub(b)

			case "*":
				got = a.Mul(b)

			case "abs":
				got = a.Abs()
			}

			if !got.Equal(want) {
				t.Errorf("want %v %v %v = %v; got %v", tc.a, tc.op, tc.b, want, got)
			}
		})
	}
}

func TestComparison(t *testing.T) {
	tt := []struct {
		a    string
		op   string
		b    string
		want bool
	}{
		{"1", "==", "1", true},
		{"0.1", "==", "0.1", true},
		{"1", "==", "2", false},
		{"2", "==", "1", false},
		{"0.1", "==", "0.2", false},
		{"0.2", "==", "0.1", false},
		{"0.1", "==", "0.01", false},
		{"0.01", "==", "0.1", false},

		{"1", "<", "1", false},
		{"0.1", "<", "0.1", false},
		{"1", "<", "2", true},
		{"2", "<", "1", false},
		{"0.1", "<", "0.2", true},
		{"0.2", "<", "0.1", false},
		{"0.1", "<", "0.01", false},
		{"0.01", "<", "0.1", true},

		{"1", "<=", "1", true},
		{"0.1", "<=", "0.1", true},
		{"1", "<=", "2", true},
		{"2", "<=", "1", false},
		{"0.1", "<=", "0.2", true},
		{"0.2", "<=", "0.1", false},
		{"0.1", "<=", "0.01", false},
		{"0.01", "<=", "0.1", true},

		{"1", ">", "1", false},
		{"0.1", ">", "0.1", false},
		{"1", ">", "2", false},
		{"2", ">", "1", true},
		{"0.1", ">", "0.2", false},
		{"0.2", ">", "0.1", true},
		{"0.1", ">", "0.01", true},
		{"0.01", ">", "0.1", false},

		{"1", ">=", "1", true},
		{"0.1", ">=", "0.1", true},
		{"1", ">=", "2", false},
		{"2", ">=", "1", true},
		{"0.1", ">=", "0.2", false},
		{"0.2", ">=", "0.1", true},
		{"0.1", ">=", "0.01", true},
		{"0.01", ">=", "0.1", false},
	}
	for i, tc := range tt {
		t.Run("test["+strconv.Itoa(i)+"]", func(t *testing.T) {
			a := errsx.Must(amount.New(tc.a))
			b := errsx.Must(amount.New(tc.b))

			var got bool
			switch tc.op {
			case "==":
				got = a.Equal(b)

			case "<":
				got = a.Less(b)

			case "<=":
				got = a.LessEqual(b)

			case ">":
				got = a.Greater(b)

			case ">=":
				got = a.GreaterEqual(b)

			default:
				t.Fatalf("unknown operator: %v", tc.op)
			}

			if got != tc.want {
				t.Errorf("want %v %v %v = %v; got %v", tc.a, tc.op, tc.b, tc.want, got)
			}
		})
	}
}

func TestAllocateBetween(t *testing.T) {
	tt := []struct {
		value    string
		portions int
		want     []string
		split    int
	}{
		{"0.567", -1, []string{}, 0},
		{"0.567", 0, []string{}, 0},
		{"0.567", 1, []string{"0.567"}, 0},
		{"0.567", 2, []string{"0.284", "0.283"}, 1},
		{"0.567", 3, []string{"0.189", "0.189", "0.189"}, 0},
		{"0.567", 4, []string{"0.142", "0.142", "0.142", "0.141"}, 3},
		{"0.567", 5, []string{"0.114", "0.114", "0.113", "0.113", "0.113"}, 2},
		{"0.567", 6, []string{"0.095", "0.095", "0.095", "0.094", "0.094", "0.094"}, 3},
		{"0.567", 7, []string{"0.081", "0.081", "0.081", "0.081", "0.081", "0.081", "0.081"}, 0},
		{"0.567", 8, []string{"0.071", "0.071", "0.071", "0.071", "0.071", "0.071", "0.071", "0.070"}, 7},
		{"0.567", 9, []string{"0.063", "0.063", "0.063", "0.063", "0.063", "0.063", "0.063", "0.063", "0.063"}, 0},
		{"0.567", 10, []string{"0.057", "0.057", "0.057", "0.057", "0.057", "0.057", "0.057", "0.056", "0.056", "0.056"}, 7},

		{"3", -1, []string{}, 0},
		{"3", 0, []string{}, 0},
		{"3", 1, []string{"3"}, 0},
		{"3", 2, []string{"2", "1"}, 1},
		{"3", 3, []string{"1", "1", "1"}, 0},
		{"3", 4, []string{"1", "1", "1", "0"}, 3},
		{"3", 5, []string{"1", "1", "1", "0", "0"}, 3},
		{"3", 6, []string{"1", "1", "1", "0", "0", "0"}, 3},
		{"3", 7, []string{"1", "1", "1", "0", "0", "0", "0"}, 3},
		{"3", 8, []string{"1", "1", "1", "0", "0", "0", "0", "0"}, 3},
		{"3", 9, []string{"1", "1", "1", "0", "0", "0", "0", "0", "0"}, 3},
		{"3", 10, []string{"1", "1", "1", "0", "0", "0", "0", "0", "0", "0"}, 3},

		{"10.00", -1, []string{}, 0},
		{"10.00", 0, []string{}, 0},
		{"10.00", 1, []string{"10.00"}, 0},
		{"10.00", 2, []string{"5.00", "5.00"}, 0},
		{"10.00", 3, []string{"3.34", "3.33", "3.33"}, 1},
		{"10.00", 4, []string{"2.50", "2.50", "2.50", "2.50"}, 0},
		{"10.00", 5, []string{"2.00", "2.00", "2.00", "2.00", "2.00"}, 0},
		{"10.00", 6, []string{"1.67", "1.67", "1.67", "1.67", "1.66", "1.66"}, 4},
		{"10.00", 7, []string{"1.43", "1.43", "1.43", "1.43", "1.43", "1.43", "1.42"}, 6},
		{"10.00", 8, []string{"1.25", "1.25", "1.25", "1.25", "1.25", "1.25", "1.25", "1.25"}, 0},
		{"10.00", 9, []string{"1.12", "1.11", "1.11", "1.11", "1.11", "1.11", "1.11", "1.11", "1.11"}, 1},
		{"10.00", 10, []string{"1.00", "1.00", "1.00", "1.00", "1.00", "1.00", "1.00", "1.00", "1.00", "1.00"}, 0},

		{"105.25", -1, []string{}, 0},
		{"105.25", 0, []string{}, 0},
		{"105.25", 1, []string{"105.25"}, 0},
		{"105.25", 2, []string{"52.63", "52.62"}, 1},
		{"105.25", 3, []string{"35.09", "35.08", "35.08"}, 1},
		{"105.25", 4, []string{"26.32", "26.31", "26.31", "26.31"}, 1},
		{"105.25", 5, []string{"21.05", "21.05", "21.05", "21.05", "21.05"}, 0},
		{"105.25", 6, []string{"17.55", "17.54", "17.54", "17.54", "17.54", "17.54"}, 1},
		{"105.25", 7, []string{"15.04", "15.04", "15.04", "15.04", "15.03", "15.03", "15.03"}, 4},
		{"105.25", 8, []string{"13.16", "13.16", "13.16", "13.16", "13.16", "13.15", "13.15", "13.15"}, 5},
		{"105.25", 9, []string{"11.70", "11.70", "11.70", "11.70", "11.69", "11.69", "11.69", "11.69", "11.69"}, 4},
		{"105.25", 10, []string{"10.53", "10.53", "10.53", "10.53", "10.53", "10.52", "10.52", "10.52", "10.52", "10.52"}, 5},
	}
	for _, tc := range tt {
		name := fmt.Sprintf("%v portions of %v", tc.portions, tc.value)

		t.Run(name, func(t *testing.T) {
			if tc.portions >= 0 && tc.portions != len(tc.want) {
				t.Fatalf("test wants %v portions but describes %v expectations", tc.portions, len(tc.want))
			}

			amts, split := errsx.Must(amount.New(tc.value)).AllocateBetween(tc.portions)

			if len(amts) != len(tc.want) {
				t.Errorf("want %v amounts, got %v", len(tc.want), len(amts))
			} else {
				var total amount.Amount
				for i, got := range amts {
					if got.String() != tc.want[i] {
						t.Errorf("want portion[%v] to be %v, got %v", i, tc.want[i], got)
					}

					total = total.Add(got)
				}

				if tc.portions > 0 && total.String() != tc.value {
					t.Errorf("want total of portions to be %v, got %v", tc.value, total)
				}
			}

			if split != tc.split {
				t.Errorf("want split index to be %v, got %v", tc.split, split)
			}
		})
	}
}

func TestAbs(t *testing.T) {
	tt := []struct {
		name  string
		value string
		want  string
	}{
		{"positive", "1.075 kg", "1.075 kg"},
		{"negative", "-1.075 kg", "1.075 kg"},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			got := errsx.Must(amount.New(tc.value)).Abs()
			if got.String() != tc.want {
				t.Errorf("want abs value of %v, got %v", tc.want, got)
			}
		})
	}
}

func TestRounding(t *testing.T) {
	tt := []struct {
		name   string
		value  string
		places int
		mode   amount.RoundingMode
		want   string
	}{
		{"negative places, treat as zero", "1.075 kg", -3, amount.Truncate, "1 kg"},

		{"truncate, same places, no change", "1.075 kg", 3, amount.Truncate, "1.075 kg"},
		{"truncate, larger places, pad zeros", "1.075 kg", 5, amount.Truncate, "1.07500 kg"},
		{"truncate, places 1", "1.075 kg", 1, amount.Truncate, "1.0 kg"},
		{"truncate, places 1", "1.055 kg", 1, amount.Truncate, "1.0 kg"},
		{"truncate, places 1", "1.0558 kg", 1, amount.Truncate, "1.0 kg"},
		{"truncate, places 2", "294.934999999999995", 2, amount.Truncate, "294.93"},
		{"truncate, places 1, negative", "-1.075 kg", 1, amount.Truncate, "-1.0 kg"},
		{"truncate, places 1, negative", "-1.055 kg", 1, amount.Truncate, "-1.0 kg"},
		{"truncate, places 1, negative", "-1.0558 kg", 1, amount.Truncate, "-1.0 kg"},
		{"truncate, places 2, negative", "-294.934999999999995", 2, amount.Truncate, "-294.93"},

		{"half away from zero, larger places, pad zeros", "1.075 kg", 5, amount.HalfAwayFromZero, "1.07500 kg"},
		{"half away from zero, places 2", "1.075 kg", 2, amount.HalfAwayFromZero, "1.08 kg"},
		{"half away from zero, places 2", "1.078 kg", 2, amount.HalfAwayFromZero, "1.08 kg"},
		{"half away from zero, places 2", "1.0788 kg", 2, amount.HalfAwayFromZero, "1.08 kg"},
		{"half away from zero, places 2", "294.934999999999995", 2, amount.HalfAwayFromZero, "294.93"},
		{"half away from zero, places 2", "294.935999999999995", 2, amount.HalfAwayFromZero, "294.94"},
		{"half away from zero, places 2", "294.936999999999995", 2, amount.HalfAwayFromZero, "294.94"},
		{"half away from zero, places 2, negative", "-1.075 kg", 2, amount.HalfAwayFromZero, "-1.08 kg"},
		{"half away from zero, places 2, negative", "-1.078 kg", 2, amount.HalfAwayFromZero, "-1.08 kg"},
		{"half away from zero, places 2, negative", "-1.0788 kg", 2, amount.HalfAwayFromZero, "-1.08 kg"},
		{"half away from zero, places 2, negative", "-294.934999999999995", 2, amount.HalfAwayFromZero, "-294.93"},
		{"half away from zero, places 2, negative", "-294.935999999999995", 2, amount.HalfAwayFromZero, "-294.94"},
		{"half away from zero, places 2, negative", "-294.936999999999995", 2, amount.HalfAwayFromZero, "-294.94"},

		{"half towards zero, larger places, pad zeros", "1.075 kg", 5, amount.HalfTowardsZero, "1.07500 kg"},
		{"half towards zero, places 2", "1.070 kg", 2, amount.HalfTowardsZero, "1.07 kg"},
		{"half towards zero, places 2", "1.075 kg", 2, amount.HalfTowardsZero, "1.07 kg"},
		{"half towards zero, places 2", "1.078 kg", 2, amount.HalfTowardsZero, "1.08 kg"},
		{"half towards zero, places 2", "1.0788 kg", 2, amount.HalfTowardsZero, "1.08 kg"},
		{"half towards zero, places 2", "294.934999999999995", 2, amount.HalfTowardsZero, "294.93"},
		{"half towards zero, places 2", "294.935999999999995", 2, amount.HalfTowardsZero, "294.93"},
		{"half towards zero, places 2", "294.936999999999995", 2, amount.HalfTowardsZero, "294.94"},
		{"half towards zero, places 2, negative", "-1.075 kg", 2, amount.HalfTowardsZero, "-1.07 kg"},
		{"half towards zero, places 2, negative", "-0.06 kg", 1, amount.HalfTowardsZero, "-0.1 kg"},
		{"half towards zero, places 2, negative", "-0.068 kg", 1, amount.HalfTowardsZero, "-0.1 kg"},
		{"half towards zero, places 2, negative", "-294.934999999999995", 2, amount.HalfTowardsZero, "-294.93"},
		{"half towards zero, places 2, negative", "-294.935999999999995", 2, amount.HalfTowardsZero, "-294.93"},
		{"half towards zero, places 2, negative", "-294.936999999999995", 2, amount.HalfTowardsZero, "-294.94"},

		{"half to even zero, larger places, pad zeros", "1.075 kg", 5, amount.HalfToEven, "1.07500 kg"},
		{"half to even, places 2", "1.065 kg", 2, amount.HalfToEven, "1.06 kg"},
		{"half to even, places 2", "1.085 kg", 2, amount.HalfToEven, "1.08 kg"},
		{"half to even, places 2", "1.088 kg", 2, amount.HalfToEven, "1.09 kg"},
		{"half to even, places 2", "1.0888 kg", 2, amount.HalfToEven, "1.09 kg"},
		{"half to even, places 2", "294.934999999999995", 2, amount.HalfToEven, "294.93"},
		{"half to even, places 2", "294.935999999999995", 2, amount.HalfToEven, "294.94"},
		{"half to even, places 2", "294.936999999999995", 2, amount.HalfToEven, "294.94"},
		{"half to even, places 2", "294.944999999999995", 2, amount.HalfToEven, "294.94"},
		{"half to even, places 2", "294.945999999999995", 2, amount.HalfToEven, "294.94"},
		{"half to even, places 2", "294.946999999999995", 2, amount.HalfToEven, "294.95"},
		{"half to even, places 2, negative", "-1.065 kg", 2, amount.HalfToEven, "-1.06 kg"},
		{"half to even, places 2, negative", "-1.085 kg", 2, amount.HalfToEven, "-1.08 kg"},
		{"half to even, places 2, negative", "-1.088 kg", 2, amount.HalfToEven, "-1.09 kg"},
		{"half to even, places 2, negative", "-1.0888 kg", 2, amount.HalfToEven, "-1.09 kg"},
		{"half to even, places 2, negative", "-294.934999999999995", 2, amount.HalfToEven, "-294.93"},
		{"half to even, places 2, negative", "-294.935999999999995", 2, amount.HalfToEven, "-294.94"},
		{"half to even, places 2, negative", "-294.936999999999995", 2, amount.HalfToEven, "-294.94"},
		{"half to even, places 2, negative", "-294.944999999999995", 2, amount.HalfToEven, "-294.94"},
		{"half to even, places 2, negative", "-294.945999999999995", 2, amount.HalfToEven, "-294.94"},
		{"half to even, places 2, negative", "-294.946999999999995", 2, amount.HalfToEven, "-294.95"},

		{"half to odd zero, larger places, pad zeros", "1.075 kg", 5, amount.HalfToOdd, "1.07500 kg"},
		{"half to odd, places 2", "1.065 kg", 2, amount.HalfToOdd, "1.07 kg"},
		{"half to odd, places 2", "1.075 kg", 2, amount.HalfToOdd, "1.07 kg"},
		{"half to odd, places 2", "1.078 kg", 2, amount.HalfToOdd, "1.08 kg"},
		{"half to odd, places 2", "1.0788 kg", 2, amount.HalfToOdd, "1.08 kg"},
		{"half to odd, places 2", "294.934999999999995", 2, amount.HalfToOdd, "294.93"},
		{"half to odd, places 2", "294.935999999999995", 2, amount.HalfToOdd, "294.93"},
		{"half to odd, places 2", "294.936999999999995", 2, amount.HalfToOdd, "294.94"},
		{"half to odd, places 2", "294.944999999999995", 2, amount.HalfToOdd, "294.94"},
		{"half to odd, places 2", "294.945999999999995", 2, amount.HalfToOdd, "294.95"},
		{"half to odd, places 2", "294.946999999999995", 2, amount.HalfToOdd, "294.95"},
		{"half to odd, places 2, negative", "-1.065 kg", 2, amount.HalfToOdd, "-1.07 kg"},
		{"half to odd, places 2, negative", "-1.075 kg", 2, amount.HalfToOdd, "-1.07 kg"},
		{"half to odd, places 2, negative", "-1.078 kg", 2, amount.HalfToOdd, "-1.08 kg"},
		{"half to odd, places 2, negative", "-1.0788 kg", 2, amount.HalfToOdd, "-1.08 kg"},
		{"half to odd, places 2, negative", "-294.934999999999995", 2, amount.HalfToOdd, "-294.93"},
		{"half to odd, places 2, negative", "-294.935999999999995", 2, amount.HalfToOdd, "-294.93"},
		{"half to odd, places 2, negative", "-294.936999999999995", 2, amount.HalfToOdd, "-294.94"},
		{"half to odd, places 2, negative", "-294.944999999999995", 2, amount.HalfToOdd, "-294.94"},
		{"half to odd, places 2, negative", "-294.945999999999995", 2, amount.HalfToOdd, "-294.95"},
		{"half to odd, places 2, negative", "-294.946999999999995", 2, amount.HalfToOdd, "-294.95"},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			got := errsx.Must(amount.New(tc.value)).Round(tc.places, tc.mode)
			want := errsx.Must(amount.New(tc.want))
			if !got.Equal(want) {
				t.Errorf("want rounded value of %v, got %v (places: %v, mode: %v)", tc.want, got, tc.places, tc.mode)
			}
		})
	}
}

func TestString(t *testing.T) {
	tt := []struct {
		value string
		want  string
	}{
		{"-0.1", "-0.1"},
		{"0.1", "0.1"},
		{"+0.1", "0.1"},
		{"100", "100"},
		{"100.000", "100.000"},
		{"100.020", "100.020"},
		{"00003240.0001000", "3240.0001000"},
	}
	for i, tc := range tt {
		t.Run("test["+strconv.Itoa(i)+"]", func(t *testing.T) {
			got := errsx.Must(amount.New(tc.value)).String()
			if got != tc.want {
				t.Errorf("want string %q; got %q", tc.want, got)
			}
		})
	}
}

func TestFormat(t *testing.T) {
	tt := []struct {
		value  string
		places int
		want   string
	}{
		{"-0.1", 2, "-0.10"},
		{"0.1", 2, "0.10"},
		{"+0.1", 2, "0.10"},
		{"100", 2, "100.00"},
		{"100.000", 2, "100.00"},
		{"100.020", 2, "100.02"},
		{"00003240.0001000", 2, "3240.0001"},
	}
	for i, tc := range tt {
		t.Run("test["+strconv.Itoa(i)+"]", func(t *testing.T) {
			got := errsx.Must(amount.New(tc.value)).Format(tc.places)
			if got != tc.want {
				t.Errorf("want format %q; got %q", tc.want, got)
			}
		})
	}
}

func TestPanics(t *testing.T) {
	tt := []struct {
		name string
		a    string
		op   string
		b    string
	}{
		{"different units: add", "1 a", "+", "1 b"},
		{"different units: sub", "1 a", "-", "1 b"},
		{"different units: mul", "1 a", "*", "1 b"},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r == nil {
					t.Error("want a panic, got nil")
				}
			}()

			a := errsx.Must(amount.New(tc.a))

			if tc.op != "" {
				b := errsx.Must(amount.New(tc.b))

				switch tc.op {
				case "+":
					a.Add(b)

				case "-":
					a.Sub(b)

				case "*":
					a.Mul(b)
				}
			}
		})
	}
}

func TestScanner(t *testing.T) {
	tt := []struct {
		name        string
		src         any
		value       string
		places      int
		shouldError bool
	}{
		{"int64 value", int64(12), "0", 0, true},
		{"float64 value", float64(12.93), "0", 0, true},
		{"bool value", true, "0", 0, true},
		{"string value", "12.93 kg", "12.93 kg", 2, false},
		{"[]byte value", []byte("12.93"), "12.93", 2, false},
		{"time.Time value", time.Now(), "0", 0, true},
		{"nil value", nil, "0", 0, false},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			var amt amount.Amount
			err := amt.Scan(tc.src)
			if tc.shouldError && err == nil {
				t.Error("want error, got nil")
			}
			if tc.value != amt.String() {
				t.Errorf("want string value %q, got %q", tc.value, amt)
			}
			if tc.places != amt.Places() {
				t.Errorf("want places value %d, got %d", tc.places, amt.Places())
			}
		})
	}
}

func TestValuer(t *testing.T) {
	tt := []struct {
		name   string
		amount string
		value  string
		places int
	}{
		{"empty", "", "0", 0},
		{"zero", "0", "0", 0},
		{"zero normalized", "0.00", "0.00", 2},
		{"valid decimal point", "13.453 kg", "13.453 kg", 3},
		{"valid negative decimal point", "-13.453", "-13.453", 3},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			var amt amount.Amount
			var err error
			if tc.amount != "" {
				amt = errsx.Must(amount.New(tc.amount))
			}

			value, err := amt.Value()
			if err != nil {
				t.Fatal(err)
			}

			v, ok := value.(string)
			if !ok {
				t.Fatalf("want string, got %T", value)
			}

			if tc.value != v {
				t.Errorf("want string value %q, got %q", tc.value, v)
			}
			if tc.places != amt.Places() {
				t.Errorf("want places value %d, got %d", tc.places, amt.Places())
			}
		})
	}
}

func TestMarshaler(t *testing.T) {
	tt := []struct {
		name  string
		value string
		json  string
	}{
		{"empty", "", `{"amount":"0"}`},
		{"zero", "0", `{"amount":"0"}`},
		{"zero with places", "0.00", `{"amount":"0.00"}`},
		{"valid decimal point", "13.453 kg", `{"amount":"13.453 kg"}`},
		{"valid negative decimal point", "-13.453", `{"amount":"-13.453"}`},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			var amt amount.Amount
			var err error
			if tc.value == "" {
				amt = amount.Amount{}
			} else {
				amt = errsx.Must(amount.New(tc.value))
			}
			j, err := json.Marshal(map[string]amount.Amount{"amount": amt})
			if err != nil {
				t.Fatal(err)
			}
			if tc.json != string(j) {
				t.Errorf("want json value %v, got %v", tc.json, string(j))
			}
		})
	}
}

func TestUnmarshaler(t *testing.T) {
	tt := []struct {
		name  string
		value string
		json  string
	}{
		{"empty", "0", `{"amount":""}`},
		{"zero", "0", `{"amount":"0"}`},
		{"zero with places", "0.00", `{"amount":"0.00"}`},
		{"valid decimal point", "13.453 kg", `{"amount":"13.453 kg"}`},
		{"valid negative decimal point", "-13.453", `{"amount":"-13.453"}`},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			data := map[string]amount.Amount{}
			err := json.Unmarshal([]byte(tc.json), &data)
			if err != nil {
				t.Fatal(err)
			}
			if tc.value != data["amount"].String() {
				t.Errorf("want amount value %v, got %v", tc.value, data["amount"])
			}
		})
	}
}
