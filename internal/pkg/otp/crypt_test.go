package otp_test

import (
	"testing"

	"github.com/polyscone/tofu/internal/pkg/otp"
)

func TestNewKey(t *testing.T) {
	tt := []struct {
		name   string
		alg    otp.Alg
		length int
	}{
		{"newhash for sha1", otp.SHA1, 20},
		{"newhash for sha512", otp.SHA512, 64},
	}
	for _, tc := range tt {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			key, err := otp.NewKey(nil, tc.alg)
			if err != nil {
				t.Errorf("want <nil>; got %q", err)
			}
			if want, got := tc.length, len(key); want != got {
				t.Errorf("want %v; got %v", want, got)
			}
		})
	}
}
