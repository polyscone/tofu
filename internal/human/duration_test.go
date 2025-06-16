package human_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/polyscone/tofu/internal/human"
)

func TestDuration(t *testing.T) {
	tt := []struct {
		input time.Duration
		want  string
	}{
		{0, "0 seconds"},
		{1 * time.Nanosecond, "0 seconds"},
		{1 * time.Microsecond, "0 seconds"},
		{1 * time.Millisecond, "0 seconds"},
		{1 * time.Second, "1 second"},
		{1 * time.Minute, "1 minute"},
		{1 * time.Hour, "1 hour"},
		{24 * time.Hour, "1 day"},
		{365 * 24 * time.Hour, "1 year"},

		{-1 * time.Nanosecond, "0 seconds"},
		{-1 * time.Microsecond, "0 seconds"},
		{-1 * time.Millisecond, "0 seconds"},
		{-1 * time.Second, "-1 second"},
		{-1 * time.Minute, "-1 minute"},
		{-1 * time.Hour, "-1 hour"},
		{-24 * time.Hour, "-1 day"},
		{-365 * 24 * time.Hour, "-1 year"},
	}
	for i, tc := range tt {
		name := fmt.Sprintf("%v: %v", i, tc.input)

		t.Run(name, func(t *testing.T) {
			got := human.Duration(tc.input)

			if tc.want != got {
				t.Errorf("want %v; got %v", tc.want, got)
			}
		})
	}
}

func TestDurationStat(t *testing.T) {
	tt := []struct {
		input time.Duration
		want  string
	}{
		{0, "0 s"},
		{1 * time.Nanosecond, "1 ns"},
		{1 * time.Microsecond, "1 µs"},
		{1 * time.Millisecond, "1 ms"},
		{1 * time.Second, "1 s"},
		{1 * time.Minute, "1 m"},
		{1 * time.Hour, "1 h"},
		{24 * time.Hour, "24 h"},
		{365 * 24 * time.Hour, "8760 h"},

		{-1 * time.Nanosecond, "-1 ns"},
		{-1 * time.Microsecond, "-1 µs"},
		{-1 * time.Millisecond, "-1 ms"},
		{-1 * time.Second, "-1 s"},
		{-1 * time.Minute, "-1 m"},
		{-1 * time.Hour, "-1 h"},
		{-24 * time.Hour, "-24 h"},
		{-365 * 24 * time.Hour, "-8760 h"},
	}
	for i, tc := range tt {
		name := fmt.Sprintf("%v: %v", i, tc.input)

		t.Run(name, func(t *testing.T) {
			got := human.DurationStat(tc.input)

			if tc.want != got {
				t.Errorf("want %v; got %v", tc.want, got)
			}
		})
	}
}
