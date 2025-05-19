package i18n_test

import (
	"strings"
	"testing"

	"github.com/polyscone/tofu/internal/i18n"
)

func TestScanner(t *testing.T) {
	s := i18n.NewScanner()

	tt := []struct {
		name  string
		input string
		want  []i18n.Token
	}{
		{"eof", "", []i18n.Token{{i18n.KindEOF, ""}}},

		{"integer", "{123}", []i18n.Token{
			{i18n.KindEnterExpr, "{"},
			{i18n.KindInt, "123"},
			{i18n.KindLeaveExpr, "}"},
		}},

		{"float", "{123.123}", []i18n.Token{
			{i18n.KindEnterExpr, "{"},
			{i18n.KindFloat, "123.123"},
			{i18n.KindLeaveExpr, "}"},
		}},

		{"single quote string", "{'Hello, World!'}", []i18n.Token{
			{i18n.KindEnterExpr, "{"},
			{i18n.KindString, "Hello, World!"},
			{i18n.KindLeaveExpr, "}"},
		}},
		{"single quote string with escaped delim", `{'Hello, \'World\'!'}`, []i18n.Token{
			{i18n.KindEnterExpr, "{"},
			{i18n.KindString, `Hello, 'World'!`},
			{i18n.KindLeaveExpr, "}"},
		}},
		{"single quote string with escaped escape", `{'Hello, \\ World!'}`, []i18n.Token{
			{i18n.KindEnterExpr, "{"},
			{i18n.KindString, `Hello, \ World!`},
			{i18n.KindLeaveExpr, "}"},
		}},

		{"double quote string", `{"Hello, World!"}`, []i18n.Token{
			{i18n.KindEnterExpr, "{"},
			{i18n.KindString, "Hello, World!"},
			{i18n.KindLeaveExpr, "}"},
		}},
		{"double quote string with escaped delim", `{"Hello, \"World\"!"}`, []i18n.Token{
			{i18n.KindEnterExpr, "{"},
			{i18n.KindString, `Hello, "World"!`},
			{i18n.KindLeaveExpr, "}"},
		}},
		{"double quote string with escaped escape", `{"Hello, \\ World!"}`, []i18n.Token{
			{i18n.KindEnterExpr, "{"},
			{i18n.KindString, `Hello, \ World!`},
			{i18n.KindLeaveExpr, "}"},
		}},

		{"backtick string", "{`Hello, World!`}", []i18n.Token{
			{i18n.KindEnterExpr, "{"},
			{i18n.KindString, "Hello, World!"},
			{i18n.KindLeaveExpr, "}"},
		}},
		{"backtick quote string with escaped delim", "{`Hello, \\`World\\`!`}", []i18n.Token{
			{i18n.KindEnterExpr, "{"},
			{i18n.KindString, "Hello, `World`!"},
			{i18n.KindLeaveExpr, "}"},
		}},
		{"backtick quote string with escaped escape", "{`Hello, \\\\ World!`}", []i18n.Token{
			{i18n.KindEnterExpr, "{"},
			{i18n.KindString, `Hello, \ World!`},
			{i18n.KindLeaveExpr, "}"},
		}},

		{"text", "Foo bar baz", []i18n.Token{{i18n.KindText, "Foo bar baz"}}},
		{"text with expr start and end", "Foo {bar} baz", []i18n.Token{
			{i18n.KindText, "Foo "},
			{i18n.KindEnterExpr, "{"},
			{i18n.KindIdent, "bar"},
			{i18n.KindLeaveExpr, "}"},
			{i18n.KindText, " baz"},
		}},
		{"text with expr start and end and unexpected", "Foo {{{ bar }}} baz", []i18n.Token{
			{i18n.KindText, "Foo "},
			{i18n.KindEnterExpr, "{"},
			{i18n.KindUnexpected, "{{"},
			{i18n.KindIdent, "bar"},
			{i18n.KindLeaveExpr, "}"},
			{i18n.KindText, "}} baz"},
		}},
		{
			"text that contains operators and literals",
			"or and ! , : () [] + - * / % == != > >= < <= {123} or and ! , : () [] + - * / % == != > >= < <=",
			[]i18n.Token{
				{i18n.KindText, "or and ! , : () [] + - * / % == != > >= < <= "},
				{i18n.KindEnterExpr, "{"},
				{i18n.KindInt, "123"},
				{i18n.KindLeaveExpr, "}"},
				{i18n.KindText, " or and ! , : () [] + - * / % == != > >= < <="},
			},
		},
		{"text with escaped expr start", `Foo \{bar} baz`, []i18n.Token{{i18n.KindText, "Foo {bar} baz"}}},
		{"text with escaped expr and escaped backslash", `Foo \\{bar} baz`, []i18n.Token{
			{i18n.KindText, `Foo \`},
			{i18n.KindEnterExpr, "{"},
			{i18n.KindIdent, "bar"},
			{i18n.KindLeaveExpr, "}"},
			{i18n.KindText, " baz"},
		}},

		{"or", "{or}", []i18n.Token{
			{i18n.KindEnterExpr, "{"},
			{i18n.KindOr, "or"},
			{i18n.KindLeaveExpr, "}"},
		}},
		{"and", "{and}", []i18n.Token{
			{i18n.KindEnterExpr, "{"},
			{i18n.KindAnd, "and"},
			{i18n.KindLeaveExpr, "}"},
		}},

		{"bang", "{!}", []i18n.Token{
			{i18n.KindEnterExpr, "{"},
			{i18n.KindBang, "!"},
			{i18n.KindLeaveExpr, "}"},
		}},
		{"comma", "{,}", []i18n.Token{
			{i18n.KindEnterExpr, "{"},
			{i18n.KindComma, ","},
			{i18n.KindLeaveExpr, "}"},
		}},
		{"colon", "{:}", []i18n.Token{
			{i18n.KindEnterExpr, "{"},
			{i18n.KindColon, ":"},
			{i18n.KindLeaveExpr, "}"},
		}},

		{"left paren", "{(}", []i18n.Token{
			{i18n.KindEnterExpr, "{"},
			{i18n.KindLParen, "("},
			{i18n.KindLeaveExpr, "}"},
		}},
		{"right paren", "{)}", []i18n.Token{
			{i18n.KindEnterExpr, "{"},
			{i18n.KindRParen, ")"},
			{i18n.KindLeaveExpr, "}"},
		}},

		{"left brack", "{[}", []i18n.Token{
			{i18n.KindEnterExpr, "{"},
			{i18n.KindLBrack, "["},
			{i18n.KindLeaveExpr, "}"},
		}},
		{"right brack", "{]}", []i18n.Token{
			{i18n.KindEnterExpr, "{"},
			{i18n.KindRBrack, "]"},
			{i18n.KindLeaveExpr, "}"},
		}},

		{"plus", "{+}", []i18n.Token{
			{i18n.KindEnterExpr, "{"},
			{i18n.KindPlus, "+"},
			{i18n.KindLeaveExpr, "}"},
		}},
		{"minus", "{-}", []i18n.Token{
			{i18n.KindEnterExpr, "{"},
			{i18n.KindMinus, "-"},
			{i18n.KindLeaveExpr, "}"},
		}},
		{"asterisk", "{*}", []i18n.Token{
			{i18n.KindEnterExpr, "{"},
			{i18n.KindAsterisk, "*"},
			{i18n.KindLeaveExpr, "}"},
		}},
		{"slash", "{/}", []i18n.Token{
			{i18n.KindEnterExpr, "{"},
			{i18n.KindSlash, "/"},
			{i18n.KindLeaveExpr, "}"},
		}},
		{"percent", "{%}", []i18n.Token{
			{i18n.KindEnterExpr, "{"},
			{i18n.KindPercent, "%"},
			{i18n.KindLeaveExpr, "}"},
		}},

		{"equal", "{=}", []i18n.Token{
			{i18n.KindEnterExpr, "{"},
			{i18n.KindEqual, "="},
			{i18n.KindLeaveExpr, "}"},
		}},

		{"equal equal", "{==}", []i18n.Token{
			{i18n.KindEnterExpr, "{"},
			{i18n.KindEqualEqual, "=="},
			{i18n.KindLeaveExpr, "}"},
		}},
		{"bang equal", "{!=}", []i18n.Token{
			{i18n.KindEnterExpr, "{"},
			{i18n.KindBangEqual, "!="},
			{i18n.KindLeaveExpr, "}"},
		}},

		{"greater", "{>}", []i18n.Token{
			{i18n.KindEnterExpr, "{"},
			{i18n.KindGreater, ">"},
			{i18n.KindLeaveExpr, "}"},
		}},
		{"greater equal", "{>=}", []i18n.Token{
			{i18n.KindEnterExpr, "{"},
			{i18n.KindGreaterEqual, ">="},
			{i18n.KindLeaveExpr, "}"},
		}},

		{"less", "{<}", []i18n.Token{
			{i18n.KindEnterExpr, "{"},
			{i18n.KindLess, "<"},
			{i18n.KindLeaveExpr, "}"},
		}},
		{"less equal", "{<=}", []i18n.Token{
			{i18n.KindEnterExpr, "{"},
			{i18n.KindLessEqual, "<="},
			{i18n.KindLeaveExpr, "}"},
		}},
		{"equal greater", "{=>}", []i18n.Token{
			{i18n.KindEnterExpr, "{"},
			{i18n.KindEqualGreater, "=>"},
			{i18n.KindLeaveExpr, "}"},
		}},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			// Append an implicit EOF token to help keep tests cleaner
			if tc.want[len(tc.want)-1].Kind != i18n.KindEOF {
				tc.want = append(tc.want, i18n.Token{Kind: i18n.KindEOF})
			}

			s.Load(strings.NewReader(tc.input))

			var i int
			for {
				tok, err := s.Consume()
				if err != nil {
					t.Fatal(err)
				}

				if n := len(tc.want); n-1 < i {
					t.Fatalf("found too many tokens, want %v", n)
				}

				want := tc.want[i]
				if kind := want.Kind; kind != tok.Kind {
					t.Errorf("token %v: want kind %#q, got %#q", i, kind, tok.Kind)
				}
				if lexeme := want.Lexeme; lexeme != tok.Lexeme {
					t.Errorf("token %v: want lexeme %#q, got %#q", i, lexeme, tok.Lexeme)
				}

				if t.Failed() {
					break
				}

				i++

				if tok.Kind == i18n.KindEOF {
					break
				}
			}
		})
	}
}
