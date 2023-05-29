package gen_test

import (
	"regexp"
	"testing"

	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/gen"
)

func TestPatternGenerator(t *testing.T) {
	const (
		excludeAll      = `^[^\x00-\x{10FFFF}]$`
		unmatchable     = `a^a`
		unmatchableWant = "aa"
	)

	tt := []struct {
		name    string
		pattern string
	}{
		{"no valid runes in range", excludeAll},
		{"empty", ``},
		{"unmatchable", unmatchable},
		{"literal", `^Hello, World!$`},
		{"digits", `^[0-9]{10}$`},
		{"not digits", `^\D{10}$`},
		{"not digits range", `^[^0-9]{10}$`},
		{"not ascii", `^[^[[:ascii:]]]{10}$`},
		{"any character not newline", `^.{10}$`},
		{"any character including newline", `(?s)^.{10}$`},
		{"multiline", `(?m)^.{10}$`},
		{"single character in class", `^[a]$`},
		{"choice between single characters in classes", `^([a]|[b])$`},
		{"printable fixed repeat", `^[[:print:]]{10}$`},
		{"printable range repeat", `^[[:print:]]{5,10}$`},
		{"printable range repeat same", `^[[:print:]]{5,5}$`},
		{"printable range repeat no upper bound", `^[[:print:]]{5,}$`},
		{"greek fixed repeat", `^\p{Greek}{12}$`},
		{"greek range repeat", `^\p{Greek}{2,20}$`},
		{"graphical any amount", `^[[:graph:]]*$`},
		{"graphical at least one", `^[[:graph:]]+$`},
		{"either or", `^(a|[b-c])+$`},
		{"alternation", `^(a|[b-c]{2})+$`},
		{"quest", `^(a(b)?)?$`},
		{"capture group", `^(ab)+$`},
		{"multiple capture groups", `^(ab(c(d)))+$`},
		{"non-capture group", `^(?:ab)+$`},
		{"word boundaries", `^\b[[:upper:]]+\b$`},
		{"no word boundaries", `^\w+\B\w+ \B $`},
		{"email", `^(?i)[a-z0-9_-]+\.?[a-z0-9_-]+@[a-z0-9_-]\.(com|co\.uk|fyi|org)$`},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			re := regexp.MustCompile(tc.pattern)
			pg := errors.Must(gen.NewPatternGenerator(tc.pattern))
			s := pg.Generate()

			switch tc.pattern {
			case excludeAll:
				if s != "" {
					t.Errorf("want empty string; got %q", s)
				}

			case unmatchable:
				if s != unmatchableWant {
					t.Errorf("want %q; got %q", unmatchableWant, s)
				}

			default:
				if !re.MatchString(s) {
					t.Errorf("want %q to match pattern %#q", s, tc.pattern)
				}
			}
		})
	}
}
