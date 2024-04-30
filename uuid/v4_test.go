package uuid_test

import (
	"regexp"
	"testing"

	"github.com/polyscone/tofu/uuid"
)

func TestV4(t *testing.T) {
	const (
		v4      = 0x04
		rfc4122 = 0x02
	)

	t.Run("new", func(t *testing.T) {
		id, err := uuid.NewV4()
		if err != nil {
			t.Fatalf("want <nil>; got %q", err)
		}

		r := "(?i)^[0-9A-F]{8}-[0-9A-F]{4}-4[0-9A-F]{3}-[89AB][0-9A-F]{3}-[0-9A-F]{12}$"
		match, err := regexp.MatchString(r, id.String())
		if err != nil {
			t.Fatalf("want <nil>; got %q", err)
		}

		if want, got := true, match; want != got {
			t.Errorf("want %v; got %v", want, got)
		}
		if want, got := byte(v4), id[6]>>4; want != got {
			t.Errorf("want %v; got %v", want, got)
		}
		if want, got := byte(rfc4122), id[8]>>6; want != got {
			t.Errorf("want %v; got %v", want, got)
		}
	})

	t.Run("parse valid", func(t *testing.T) {
		ids := []string{
			"4dd3ca6d-73b9-4410-b670-d6dd952fb513",
			"3502ec33-20a5-4ae9-9fe1-a0e95a0d7e30",
			"ff881ac0-af86-4ef5-a783-9ee2309e3332",
			"af4b34ea-c29c-4bfa-ae43-119624c5858b",
			"85c40830-2812-409c-b599-8a0765680b61",
		}
		for _, id := range ids {
			u, err := uuid.Parse(id)
			if err != nil {
				t.Fatalf("want <nil>; got %q", err)
			}
			if want, got := id, u.String(); want != got {
				t.Errorf("want %q; got %q", want, got)
			}
			if !u.IsValidV4() {
				t.Error("want valid v4 UUID")
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
			{"treat nil v4 uuids as invalid", "00000000-0000-0000-0000-000000000000"},
			{"no hyphens", "17e32970968942a6aefcbaa4241cce2a"},
			{"incorrect hypen positions", "f0-155c0e3a7441-8caea9-0fdb2fc-b0558"},
			{"correct format, invalid v4", "1d0d5fac-173e-1219-119e-12ff35b9aa20"},
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
