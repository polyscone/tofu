package size_test

import (
	"testing"

	"github.com/polyscone/tofu/internal/size"
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
		{"terabyte", size.Terabyte, 1000 * 1000 * 1000 * 1000},
		{"petabyte", size.Petabyte, 1000 * 1000 * 1000 * 1000 * 1000},

		{"kibibyte", size.Kibibyte, 1024},
		{"mebibyte", size.Mebibyte, 1024 * 1024},
		{"gibibyte", size.Gibibyte, 1024 * 1024 * 1024},
		{"tebibyte", size.Tebibyte, 1024 * 1024 * 1024 * 1024},
		{"pebibyte", size.Pebibyte, 1024 * 1024 * 1024 * 1024 * 1024},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			if tc.got != tc.want {
				t.Errorf("want %v; got %v", tc.want, tc.got)
			}
		})
	}
}

func TestParseBytes(t *testing.T) {
	tt := []struct {
		name  string
		input string
		want  int
	}{
		{"bytes no suffix", "10015", 10015},
		{"bytes with suffix b", "10015 b", 10015},
		{"bytes with suffix byte", "10015 byte", 10015},
		{"bytes with suffix bytes", "10015 bytes", 10015},
		{"bytes with commas", "10,015", 10015},

		{"kilobytes with suffix kb", "1000 kb", 1000000},
		{"kilobytes with suffix kilobyte", "1000 kilobyte", 1000000},
		{"kilobytes with suffix kilobytes", "1000 kilobytes", 1000000},
		{"kilobytes with commas", "1,000 kb", 1000000},
		{"kilobytes with decimal point", "1,000.15 kb", 1000150},

		{"megabytes with suffix mb", "1000 mb", 1000000000},
		{"megabytes with suffix megabyte", "1000 megabyte", 1000000000},
		{"megabytes with suffix megabytes", "1000 megabytes", 1000000000},
		{"megabytes with commas", "1,000 mb", 1000000000},
		{"megabytes with decimal point", "1,000.15 mb", 1000150000},

		{"gigabytes with suffix gb", "1000 gb", 1000000000000},
		{"gigabytes with suffix gigabyte", "1000 gigabyte", 1000000000000},
		{"gigabytes with suffix gigabytes", "1000 gigabytes", 1000000000000},
		{"gigabytes with commas", "1,000 gb", 1000000000000},
		{"gigabytes with decimal point", "1,000.15 gb", 1000150000000},

		{"terabytes with suffix tb", "1000 tb", 1000000000000000},
		{"terabytes with suffix terabyte", "1000 terabyte", 1000000000000000},
		{"terabytes with suffix terabytes", "1000 terabytes", 1000000000000000},
		{"terabytes with commas", "1,000 tb", 1000000000000000},
		{"terabytes with decimal point", "1,000.15 tb", 1000150000000000},

		{"petabytes with suffix pb", "1000 pb", 1000000000000000000},
		{"petabytes with suffix petabyte", "1000 petabyte", 1000000000000000000},
		{"petabytes with suffix petabytes", "1000 petabytes", 1000000000000000000},
		{"petabytes with commas", "1,000 pb", 1000000000000000000},
		{"petabytes with decimal point", "1,000.15 pb", 1000150000000000000},

		{"kibibytes with suffix kib", "1000 kib", 1024000},
		{"kibibytes with suffix kibibyte", "1000 kibibyte", 1024000},
		{"kibibytes with suffix kibibytes", "1000 kibibytes", 1024000},
		{"kibibytes with commas", "1,000 kib", 1024000},
		{"kibibytes with decimal point", "1,000.15 kib", 1024153},

		{"mebibytes with suffix mib", "1000 mib", 1048576000},
		{"mebibytes with suffix megabyte", "1000 mebibyte", 1048576000},
		{"mebibytes with suffix mebibytes", "1000 mebibytes", 1048576000},
		{"mebibytes with commas", "1,000 mib", 1048576000},
		{"mebibytes with decimal point", "1,000.15 mib", 1048733286},

		{"gibibytes with suffix gib", "1000 gib", 1073741824000},
		{"gibibytes with suffix gibibyte", "1000 gibibyte", 1073741824000},
		{"gibibytes with suffix gibibytes", "1000 gibibytes", 1073741824000},
		{"gibibytes with commas", "1,000 gib", 1073741824000},
		{"gibibytes with decimal point", "1,000.15 gib", 1073902885273},

		{"tebibytes with suffix tib", "1000 tib", 1099511627776000},
		{"tebibytes with suffix tebibyte", "1000 tebibyte", 1099511627776000},
		{"tebibytes with suffix tebibytes", "1000 tebibytes", 1099511627776000},
		{"tebibytes with commas", "1,000 tib", 1099511627776000},
		{"tebibytes with decimal point", "1,000.15 tib", 1099676554520166},

		{"pebibytes with suffix pib", "1000 pib", 1125899906842624000},
		{"pebibytes with suffix pebibyte", "1000 pebibyte", 1125899906842624000},
		{"pebibytes with suffix pebibytes", "1000 pebibytes", 1125899906842624000},
		{"pebibytes with commas", "1,000 pib", 1125899906842624000},
		{"pebibytes with decimal point", "1,000.15 pib", 1126068791828650368},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			got, err := size.ParseBytes(tc.input)
			if err != nil {
				t.Fatal(err)
			}

			if tc.want != got {
				t.Errorf("want %v, got %v", tc.want, got)
			}
		})
	}
}
