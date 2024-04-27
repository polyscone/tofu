package uuid_test

import (
	"regexp"
	"testing"

	"github.com/polyscone/tofu/pkg/uuid"
)

func TestV7(t *testing.T) {
	const (
		v7      = 0x07
		rfc4122 = 0x02
	)

	t.Run("new", func(t *testing.T) {
		var last string
		for range 10_000 {
			id, err := uuid.NewV7()
			if err != nil {
				t.Fatalf("want <nil>; got %q", err)
			}

			r := "(?i)^[0-9A-F]{8}-[0-9A-F]{4}-7[0-9A-F]{3}-[89AB][0-9A-F]{3}-[0-9A-F]{12}$"
			match, err := regexp.MatchString(r, id.String())
			if err != nil {
				t.Fatalf("want <nil>; got %q", err)
			}

			if want, got := true, match; want != got {
				t.Errorf("want %v; got %v", want, got)
			}
			if want, got := byte(v7), id[6]>>4; want != got {
				t.Errorf("want %v; got %v", want, got)
			}
			if want, got := byte(rfc4122), id[8]>>6; want != got {
				t.Errorf("want %v; got %v", want, got)
			}

			if last != "" && id.String() <= last {
				t.Fatal("want newest v7 UUID to be greater than the last one")
			}

			last = id.String()
		}
	})

	t.Run("parse valid", func(t *testing.T) {
		ids := []string{
			"018dc13a-1d9f-7683-bf91-b2cb498ca4b1",
			"018dc13a-1d9f-765d-b928-183f349df13d",
			"018dc13a-1d9f-75c9-91e9-5e3302310440",
			"018dc13a-1d9f-7651-86e0-a7e7fa11979b",
			"018dc13a-1d9f-7d2b-9bfa-1e05c5c75dce",
		}
		for _, id := range ids {
			u, err := uuid.Parse(id)
			if err != nil {
				t.Fatalf("want <nil>; got %q", err)
			}
			if want, got := id, u.String(); want != got {
				t.Errorf("want %q; got %q", want, got)
			}
			if !u.IsValidV7() {
				t.Error("want valid v7 UUID")
			}
		}
	})

	t.Run("parse invalid", func(t *testing.T) {
		tt := []struct {
			name  string
			input string
		}{
			{"empty string", ""},
			{"only spaces", "        "},
			{"normal text", "foo bar baz 123"},
			{"treat nil v7 uuids as invalid", "00000000-0000-0000-0000-000000000000"},
			{"no hyphens", "17e32970968942a6aefcbaa4241cce2a"},
			{"incorrect hypen positions", "f0-155c0e3a7441-8caea9-0fdb2fc-b0558"},
			{"correct format, invalid v7", "1d0d5fac-173e-1219-119e-12ff35b9aa20"},
		}
		for _, tc := range tt {
			t.Run(tc.name, func(t *testing.T) {
				if _, err := uuid.Parse(tc.input); err == nil {
					t.Error("want error; got <nil>")
				}
			})
		}
	})
}
