package uuid_test

import (
	"regexp"
	"testing"

	"github.com/polyscone/tofu/internal/pkg/valobj/uuid"
)

const (
	v4      = 0x04
	rfc4122 = 0x02
)

func TestNewValidV4(t *testing.T) {
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
}

func TestParseValidV4(t *testing.T) {
	t.Run("non-nillable", func(t *testing.T) {
		ids := []string{
			"4dd3ca6d-73b9-4410-b670-d6dd952fb513",
			"3502ec33-20a5-4ae9-9fe1-a0e95a0d7e30",
			"ff881ac0-af86-4ef5-a783-9ee2309e3332",
			"af4b34ea-c29c-4bfa-ae43-119624c5858b",
			"85c40830-2812-409c-b599-8a0765680b61",
		}
		for _, id := range ids {
			u, err := uuid.ParseV4(id)
			if err != nil {
				t.Fatalf("want <nil>; got %q", err)
			}
			if want, got := id, u.String(); want != got {
				t.Errorf("want %q; got %q", want, got)
			}
		}
	})

	t.Run("nillable", func(t *testing.T) {
		ids := []string{
			"00000000-0000-0000-0000-000000000000",
			"4dd3ca6d-73b9-4410-b670-d6dd952fb513",
			"3502ec33-20a5-4ae9-9fe1-a0e95a0d7e30",
			"ff881ac0-af86-4ef5-a783-9ee2309e3332",
			"af4b34ea-c29c-4bfa-ae43-119624c5858b",
			"85c40830-2812-409c-b599-8a0765680b61",
		}
		for _, id := range ids {
			u, err := uuid.ParseNillableV4(id)
			if err != nil {
				t.Fatalf("want <nil>; got %q", err)
			}
			if want, got := id, u.String(); want != got {
				t.Errorf("want %q; got %q", want, got)
			}
		}
	})
}

func TestParseInvalidV4(t *testing.T) {
	t.Run("non-nillable", func(t *testing.T) {
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
			{"valid v1", "351dc63c-f387-11ec-b939-0242ac120002"},
		}
		for _, tc := range tt {
			tc := tc

			t.Run(tc.name, func(t *testing.T) {
				if _, err := uuid.ParseV4(tc.input); err == nil {
					t.Error("want error; got <nil>")
				}
			})
		}
	})

	t.Run("nillable", func(t *testing.T) {
		tt := []struct {
			name  string
			input string
		}{
			{"empty string", ""},
			{"only spaces", "        "},
			{"normal text", "foo bar baz 123"},
			{"no hyphens", "17e32970968942a6aefcbaa4241cce2a"},
			{"incorrect hypen positions", "f0-155c0e3a7441-8caea9-0fdb2fc-b0558"},
			{"correct format, invalid v4", "1d0d5fac-173e-1219-119e-12ff35b9aa20"},
			{"valid v1", "351dc63c-f387-11ec-b939-0242ac120002"},
		}
		for _, tc := range tt {
			tc := tc

			t.Run(tc.name, func(t *testing.T) {
				if _, err := uuid.ParseNillableV4(tc.input); err == nil {
					t.Error("want error; got <nil>")
				}
			})
		}
	})
}
