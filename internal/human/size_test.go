package human_test

import (
	"fmt"
	"testing"

	"github.com/polyscone/tofu/internal/human"
)

func TestSizeSI(t *testing.T) {
	tt := []struct {
		input int64
		want  string
	}{
		{0, "0 B"},
		{1, "1 B"},
		{500, "500 B"},
		{1000, "1 kB"},
		{1500, "1.5 kB"},
		{1501, "1.501 kB"},
		{1999, "1.999 kB"},
		{3000, "3 kB"},
		{4_000_000, "4 MB"},
		{5_000_000_000, "5 GB"},
		{6_000_000_000_000, "6 TB"},
		{7_000_000_000_000_000, "7 PB"},

		{-1, "-1 B"},
		{-500, "-500 B"},
		{-1000, "-1 kB"},
		{-1500, "-1.5 kB"},
		{-1501, "-1.501 kB"},
		{-1999, "-1.999 kB"},
		{-3000, "-3 kB"},
		{-4_000_000, "-4 MB"},
		{-5_000_000_000, "-5 GB"},
		{-6_000_000_000_000, "-6 TB"},
		{-7_000_000_000_000_000, "-7 PB"},
	}
	for _, tc := range tt {
		name := fmt.Sprintf("%v bytes", tc.input)

		t.Run(name, func(t *testing.T) {
			got := human.SizeSI(tc.input)

			if tc.want != got {
				t.Errorf("want %v; got %v", tc.want, got)
			}
		})
	}
}

func TestSizeIEC(t *testing.T) {
	tt := []struct {
		input int64
		want  string
	}{
		{0, "0 B"},
		{1, "1 B"},
		{500, "500 B"},
		{1023, "1023 B"},
		{1024, "1 KiB"},
		{1536, "1.5 KiB"},
		{1537, "1.5009765625 KiB"},
		{2048, "2 KiB"},
		{3072, "3 KiB"},
		{4_194_304, "4 MiB"},
		{5_368_709_120, "5 GiB"},
		{6_597_069_766_656, "6 TiB"},
		{7_881_299_347_898_368, "7 PiB"},

		{-1, "-1 B"},
		{-500, "-500 B"},
		{-1023, "-1023 B"},
		{-1024, "-1 KiB"},
		{-1536, "-1.5 KiB"},
		{-1537, "-1.5009765625 KiB"},
		{-2048, "-2 KiB"},
		{-3072, "-3 KiB"},
		{-4_194_304, "-4 MiB"},
		{-5_368_709_120, "-5 GiB"},
		{-6_597_069_766_656, "-6 TiB"},
		{-7_881_299_347_898_368, "-7 PiB"},
	}
	for _, tc := range tt {
		name := fmt.Sprintf("%v bytes", tc.input)

		t.Run(name, func(t *testing.T) {
			got := human.SizeIEC(tc.input)

			if tc.want != got {
				t.Errorf("want %v; got %v", tc.want, got)
			}
		})
	}
}
