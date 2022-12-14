package size_test

import (
	"testing"

	"github.com/polyscone/tofu/internal/pkg/size"
)

func TestSizes(t *testing.T) {
	tt := []struct {
		name string
		got  int
		want int
	}{
		{"byte", size.Byte, 1},

		{"kilobyte", size.Kilobyte, 1000},
		{"megabyte", size.Megabyte, 1000 * 1000},
		{"gigabyte", size.Gigabyte, 1000 * 1000 * 1000},

		{"kibibyte", size.Kibibyte, 1024},
		{"mebibyte", size.Mebibyte, 1024 * 1024},
		{"gibibyte", size.Gibibyte, 1024 * 1024 * 1024},
	}
	for _, tc := range tt {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			if tc.got != tc.want {
				t.Errorf("want %v; got %v", tc.want, tc.got)
			}
		})
	}
}
