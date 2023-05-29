package otp_test

import (
	"testing"

	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/otp"
)

func TestHOTPErrors(t *testing.T) {
	if _, err := otp.NewHMACBased(6, otp.SHA1); err != nil {
		t.Errorf("want <nil>; got %q", err)
	}

	if _, err := otp.NewHMACBased(5, otp.SHA1); err == nil {
		t.Error("want min digits error (6 is min) error")
	}

	if _, err := otp.NewHMACBased(6, 0); err == nil {
		t.Error("want invalid algorithm error")
	}

	hb := errors.Must(otp.NewHMACBased(6, otp.SHA1))
	if _, err := hb.Generate([]byte("1234567890123456789"), 0); err == nil {
		t.Error("want short key error")
	}

	hb = errors.Must(otp.NewHMACBased(6, otp.SHA512))
	if _, err := hb.Generate([]byte("123456789012345678901234567890123456789012345678901234567890123"), 0); err == nil {
		t.Error("want short key error")
	}
}

func TestHOTP(t *testing.T) {
	tt := []struct {
		name   string
		alg    otp.Algorithm
		digits int
		count  uint64
		hotp   string
	}{
		{"hotp digits 6, count 0", otp.SHA1, 6, 0, "755224"},
		{"hotp digits 6, count 1", otp.SHA1, 6, 1, "287082"},
		{"hotp digits 6, count 2", otp.SHA1, 6, 2, "359152"},
		{"hotp digits 6, count 3", otp.SHA1, 6, 3, "969429"},
		{"hotp digits 6, count 4", otp.SHA1, 6, 4, "338314"},
		{"hotp digits 6, count 5", otp.SHA1, 6, 5, "254676"},
		{"hotp digits 6, count 6", otp.SHA1, 6, 6, "287922"},
		{"hotp digits 6, count 7", otp.SHA1, 6, 7, "162583"},
		{"hotp digits 6, count 8", otp.SHA1, 6, 8, "399871"},
		{"hotp digits 6, count 9", otp.SHA1, 6, 9, "520489"},

		{"hotp digits 8, count 0", otp.SHA1, 8, 0, "84755224"},
		{"hotp digits 8, count 1", otp.SHA1, 8, 1, "94287082"},
		{"hotp digits 8, count 2", otp.SHA1, 8, 2, "37359152"},
		{"hotp digits 8, count 3", otp.SHA1, 8, 3, "26969429"},
		{"hotp digits 8, count 4", otp.SHA1, 8, 4, "40338314"},
		{"hotp digits 8, count 5", otp.SHA1, 8, 5, "68254676"},
		{"hotp digits 8, count 6", otp.SHA1, 8, 6, "18287922"},
		{"hotp digits 8, count 7", otp.SHA1, 8, 7, "82162583"},
		{"hotp digits 8, count 8", otp.SHA1, 8, 8, "73399871"},
		{"hotp digits 8, count 9", otp.SHA1, 8, 9, "45520489"},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			hb := errors.Must(otp.NewHMACBased(tc.digits, tc.alg))

			key := []byte("12345678901234567890") // SHA1 key
			if tc.alg == otp.SHA512 {
				key = []byte("1234567890123456789012345678901234567890123456789012345678901234") // SHA512 key
			}

			hotp := errors.Must(hb.Generate(key, tc.count))
			if want, got := tc.hotp, hotp; want != got {
				t.Errorf("want %q; got %q", want, got)
			}
		})
	}
}
