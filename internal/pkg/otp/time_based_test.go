package otp_test

import (
	"crypto/rand"
	"fmt"
	"io"
	"testing"
	"time"

	"github.com/polyscone/tofu/internal/pkg/errsx"
	"github.com/polyscone/tofu/internal/pkg/otp"
)

const defaultStep = 30 * time.Second

func TestTOTPErrors(t *testing.T) {
	if _, err := otp.NewTimeBased(6, otp.SHA1, time.Unix(0, 0), defaultStep); err != nil {
		t.Errorf("want <nil>; got %q", err)
	}

	if _, err := otp.NewTimeBased(6, otp.SHA1, time.Time{}, defaultStep); err == nil {
		t.Error("want no base time error")
	}

	if _, err := otp.NewTimeBased(6, otp.SHA1, time.Date(1880, time.November, 10, 23, 0, 0, 0, time.UTC), defaultStep); err == nil {
		t.Error("want invalid base time (before unix epoch) error")
	}

	if _, err := otp.NewTimeBased(6, otp.SHA1, time.Unix(0, 0), 29*time.Second); err == nil {
		t.Error("want min time step (30s) error")
	}

	key := errsx.Must(timeBasedKey(20))
	tb := errsx.Must(otp.NewTimeBased(6, otp.SHA1, time.Unix(0, 0), defaultStep))
	if _, err := tb.Generate(key, time.Time{}); err == nil {
		t.Error("want time not set error")
	}

	key = errsx.Must(timeBasedKey(20))
	tb = errsx.Must(otp.NewTimeBased(6, otp.SHA1, time.Unix(0, 0), defaultStep))
	if _, err := tb.Generate(key, time.Date(1880, time.November, 10, 23, 0, 0, 0, time.UTC)); err == nil {
		t.Error("want invalid time set (before unix epoch) error")
	}

	key = errsx.Must(timeBasedKey(64))
	tb = errsx.Must(otp.NewTimeBased(6, otp.SHA512, time.Unix(0, 0), defaultStep))
	if _, err := tb.Generate(key, time.Time{}); err == nil {
		t.Error("want time not set error")
	}

	key = errsx.Must(timeBasedKey(64))
	tb = errsx.Must(otp.NewTimeBased(6, otp.SHA512, time.Unix(0, 0), defaultStep))
	if _, err := tb.Generate(key, time.Date(1880, time.November, 10, 23, 0, 0, 0, time.UTC)); err == nil {
		t.Error("want invalid time set (before unix epoch) error")
	}
}

func TestTOTP(t *testing.T) {
	tt := []struct {
		name   string
		alg    otp.Algorithm
		digits int
		step   time.Duration
		time   time.Time
		totp   string
	}{
		{"totp sha1, digits 6, step 30s, time 59s", otp.SHA1, 6, defaultStep, time.Unix(59, 0), "287082"},
		{"totp sha1, digits 6, step 30s, time 1,111,111,109s", otp.SHA1, 6, defaultStep, time.Unix(1111111109, 0), "081804"},
		{"totp sha1, digits 6, step 30s, time 1,111,111,111s", otp.SHA1, 6, defaultStep, time.Unix(1111111111, 0), "050471"},
		{"totp sha1, digits 6, step 30s, time 1,234,567,890s", otp.SHA1, 6, defaultStep, time.Unix(1234567890, 0), "005924"},
		{"totp sha1, digits 6, step 30s, time 2,000,000,000s", otp.SHA1, 6, defaultStep, time.Unix(2000000000, 0), "279037"},
		{"totp sha1, digits 6, step 30s, time 20,000,000,000s", otp.SHA1, 6, defaultStep, time.Unix(20000000000, 0), "353130"},

		{"totp sha1, digits 8, step 30s, time 59s", otp.SHA1, 8, defaultStep, time.Unix(59, 0), "94287082"},
		{"totp sha1, digits 8, step 30s, time 1,111,111,109s", otp.SHA1, 8, defaultStep, time.Unix(1111111109, 0), "07081804"},
		{"totp sha1, digits 8, step 30s, time 1,111,111,111s", otp.SHA1, 8, defaultStep, time.Unix(1111111111, 0), "14050471"},
		{"totp sha1, digits 8, step 30s, time 1,234,567,890s", otp.SHA1, 8, defaultStep, time.Unix(1234567890, 0), "89005924"},
		{"totp sha1, digits 8, step 30s, time 2,000,000,000s", otp.SHA1, 8, defaultStep, time.Unix(2000000000, 0), "69279037"},
		{"totp sha1, digits 8, step 30s, time 20,000,000,000s", otp.SHA1, 8, defaultStep, time.Unix(20000000000, 0), "65353130"},

		{"totp sha512, digits 6, step 30s, time 59s", otp.SHA512, 6, defaultStep, time.Unix(59, 0), "693936"},
		{"totp sha512, digits 6, step 30s, time 1,111,111,109s", otp.SHA512, 6, defaultStep, time.Unix(1111111109, 0), "091201"},
		{"totp sha512, digits 6, step 30s, time 1,111,111,111s", otp.SHA512, 6, defaultStep, time.Unix(1111111111, 0), "943326"},
		{"totp sha512, digits 6, step 30s, time 1,234,567,890s", otp.SHA512, 6, defaultStep, time.Unix(1234567890, 0), "441116"},
		{"totp sha512, digits 6, step 30s, time 2,000,000,000s", otp.SHA512, 6, defaultStep, time.Unix(2000000000, 0), "618901"},
		{"totp sha512, digits 6, step 30s, time 20,000,000,000s", otp.SHA512, 6, defaultStep, time.Unix(20000000000, 0), "863826"},

		{"totp sha512, digits 8, step 30s, time 59s", otp.SHA512, 8, defaultStep, time.Unix(59, 0), "90693936"},
		{"totp sha512, digits 8, step 30s, time 1,111,111,109s", otp.SHA512, 8, defaultStep, time.Unix(1111111109, 0), "25091201"},
		{"totp sha512, digits 8, step 30s, time 1,111,111,111s", otp.SHA512, 8, defaultStep, time.Unix(1111111111, 0), "99943326"},
		{"totp sha512, digits 8, step 30s, time 1,234,567,890s", otp.SHA512, 8, defaultStep, time.Unix(1234567890, 0), "93441116"},
		{"totp sha512, digits 8, step 30s, time 2,000,000,000s", otp.SHA512, 8, defaultStep, time.Unix(2000000000, 0), "38618901"},
		{"totp sha512, digits 8, step 30s, time 20,000,000,000s", otp.SHA512, 8, defaultStep, time.Unix(20000000000, 0), "47863826"},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			tb := errsx.Must(otp.NewTimeBased(tc.digits, tc.alg, time.Unix(0, 0), tc.step))

			key := []byte("12345678901234567890") // SHA1 key
			if tc.alg == otp.SHA512 {
				key = []byte("1234567890123456789012345678901234567890123456789012345678901234") // SHA512 key
			}

			totp := errsx.Must(tb.Generate(key, tc.time))

			if want, got := tc.totp, totp; want != got {
				t.Errorf("want %q; got %q", want, got)
			}
		})
	}
}

func TestTOTPDefaultCheckErrors(t *testing.T) {
	tt := []struct {
		name       string
		delaySteps int
	}{
		{"totp check expect error with too many delay steps", 5},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			key := errsx.Must(timeBasedKey(20))
			totp := errsx.Must(otp.NewTimeBased(6, otp.SHA1, time.Unix(0, 0), 30*time.Second))
			if _, err := totp.Check(key, time.Now(), tc.delaySteps, ""); err == nil {
				t.Error("expected error")
			}
		})
	}
}

func TestTOTPDefaultCheck(t *testing.T) {
	now := time.Now()
	tt := []struct {
		name       string
		time       time.Time
		delaySteps int
		expected   bool
	}{
		{"totp check pass generated with current time, 1 delay step", now, 1, true},
		{"totp check pass generated one step in the past, 1 delay step", now.Add(-defaultStep), 1, true},
		{"totp check pass generated two steps in the past, 1 delay step", now.Add(-defaultStep * 2), 1, false},
		{"totp check pass generated two steps in the past, 2 delay steps", now.Add(-defaultStep * 2), 2, true},
		{"totp check pass generated one steps in the future, 1 delay step", now.Add(defaultStep), 1, true},
		{"totp check pass generated one step in the future, 0 delay steps", now.Add(defaultStep), 0, false},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			key := errsx.Must(timeBasedKey(20))
			totp := errsx.Must(otp.NewTimeBased(6, otp.SHA1, time.Unix(0, 0), 30*time.Second))
			pass := errsx.Must(totp.Generate(key, tc.time))

			ok := errsx.Must(totp.Check(key, now, tc.delaySteps, pass))

			if want, got := tc.expected, ok; want != got {
				t.Errorf("want %v; got %v", want, got)
			}
		})
	}
}

func TestTOTPDefaultCheckDuplicatePasscodes(t *testing.T) {
	key := []byte("12345678901234567890")
	now := time.Now()
	totp := errsx.Must(otp.NewTimeBased(6, otp.SHA1, time.Unix(0, 0), 30*time.Second))
	pass := errsx.Must(totp.Generate(key, now))

	ok := errsx.Must(totp.Check(key, now, 1, pass))

	// Initial check should be fine
	if want, got := true, ok; want != got {
		t.Errorf("want %v; got %v", want, got)
	}

	ok, err := totp.Check(key, now, 1, pass)
	if err == nil {
		t.Error("want duplicate error")
	}
	if want, got := false, ok; want != got {
		t.Errorf("want %v; got %v", want, got)
	}
}

func timeBasedKey(n int) ([]byte, error) {
	if n != 20 && n != 64 {
		return nil, fmt.Errorf("time based test: want key size 20 or 64; got %d", n)
	}

	b := make([]byte, n)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		return nil, fmt.Errorf("time based test: read random bytes: %w", err)
	}

	return b, nil
}
