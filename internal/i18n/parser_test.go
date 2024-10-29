package i18n_test

import (
	"strings"
	"testing"

	"github.com/polyscone/tofu/internal/i18n"
)

func TestParser(t *testing.T) {
	p := i18n.NewParser()

	tt := []struct {
		name  string
		input string
		want  string
	}{
		{"text", "Foo bar baz", `"Foo bar baz"`},
		{"text with int", "Foo {123} baz", `"Foo " 123 " baz"`},
		{"text with float", "Foo {123.999} baz", `"Foo " 123.999 " baz"`},
		{"text with string", "Foo {`Hello, World!`} baz", `"Foo " "Hello, World!" " baz"`},
		{"text with ident", "Foo {foo_bar} baz", `"Foo " foo_bar " baz"`},
		{"text with start expression", "{1} Foo bar baz", `1 " Foo bar baz"`},
		{"text with end expression", "Foo bar baz {1}", `"Foo bar baz " 1`},

		{"simple expression", "{123.123}", `123.123`},
		{"multiple expressions", "{1} {5}", `1 " " 5`},

		{"not", "{!1}", `(! 1)`},
		{"not with complex expressions", "{!(1 == 2) or !2 == 0}", `(or (! (== 1 2)) (== (! 2) 0))`},

		{"logical or", "{1 or 2}", `(or 1 2)`},
		{"logical and", "{1 and 2}", `(and 1 2)`},

		{"comparison equal", "{1 == 2}", `(== 1 2)`},
		{"comparison not equal", "{1 != 2}", `(!= 1 2)`},
		{"comparison greater", "{1 > 2}", `(> 1 2)`},
		{"comparison greater equal", "{1 >= 2}", `(>= 1 2)`},
		{"comparison less", "{1 < 2}", `(< 1 2)`},
		{"comparison less equal", "{1 <= 2}", `(<= 1 2)`},

		{"basic arithmetic", "{1 + 2 - 3 * 4 / 5 % 6}", `(- (+ 1 2) (% (/ (* 3 4) 5) 6))`},
		{"ungrouped arithmetic", "{1 + 2 * 3}", `(+ 1 (* 2 3))`},
		{"grouped arithmetic", "{(1 + 2) * 3}", `(* (+ 1 2) 3)`},

		{"indexing a variable", "{foo[1]}", `(index foo 1)`},
		{"indexing with a variable", "{foo[bar]}", `(index foo bar)`},

		{"slicing", "{foo[0:-1]}", `(slice foo 0 (- 1))`},
		{"slicing implicit end", "{foo[0:]}", `(slice foo 0)`},
		{"slicing implicit start", "{foo[:-2]}", `(slice foo 0 (- 2))`},
		{"slicing implicit start and end", "{foo[:]}", `(slice foo 0)`},
		{"slicing variables", "{foo[bar:baz]}", `(slice foo bar baz)`},

		{"slicing then indexing", "{foo[0:-1][3]}", `(index (slice foo 0 (- 1)) 3)`},
		{"indexing then slicing", "{foo[3][0:-1]}", `(slice (index foo 3) 0 (- 1))`},

		{"select from options", "{foo => (1 = 'second', _ = 'seconds')}", `(select foo (opt 1 "second") (opt _ "seconds"))`},

		{"call function no args", "{foo()}", `(call foo)`},
		{"call function one arg", "{foo(1)}", `(call foo 1)`},
		{"call function two args", "{foo(1, 'Hello, World!')}", `(call foo 1 "Hello, World!")`},
		{"call function with nested call", "{foo(1, bar())}", `(call foo 1 (call bar))`},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			src := strings.NewReader(tc.input)
			node, err := p.Parse(src)
			if err != nil {
				t.Fatal(err)
			}

			if got := i18n.SprintNode(node); tc.want != got {
				t.Errorf("\n\twant: %v\n\tgot:  %v", tc.want, got)
			}
		})
	}
}
