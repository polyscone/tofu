package i18n_test

import (
	"strings"
	"testing"

	"github.com/polyscone/tofu/internal/i18n"
)

func TestInterpreter(t *testing.T) {
	p := i18n.NewParser()
	rt := i18n.MarkdownRuntime{}
	vars := i18n.Vars{
		"int_var_1":     i18n.NewInt(1),
		"int_var_3":     i18n.NewInt(3),
		"string_var":    i18n.NewString("bar"),
		"string_var_jp": i18n.NewString("日本語"),
		"slice_var_1":   i18n.NewSlice([]i18n.Value{i18n.NewInt(1)}),
		"slice_var_3":   i18n.NewSlice([]i18n.Value{i18n.NewInt(1), i18n.NewInt(2), i18n.NewInt(3)}),
		"slice_var_5":   i18n.NewSlice([]i18n.Value{i18n.NewInt(1), i18n.NewInt(2), i18n.NewInt(3), i18n.NewInt(4), i18n.NewInt(5)}),
	}

	tt := []struct {
		name  string
		input string
		want  string
	}{
		{"bool + bool", "{(1 == 1) + (1 >= 1)} {(1 >= 1) + (1 == 1)}", `2 2`},
		{"bool + int", "{(1 == 1) + 3} {3 + (1 == 1)}", `4 4`},
		{"bool + float", "{(1 == 1) + 3.5} {3.5 + (1 == 1)}", `4.5 4.5`},
		{"bool + string", "{(1 == 1) + '2'} {'2' + (1 == 1)}", `true2 2true`},
		{"bool + slice", "{(1 == 1) + slice_var_3} {slice_var_3 + (1 == 1)}", `4 4`},

		{"int + bool", "{(1 == 1) + 3} {3 + (1 == 1)}", `4 4`},
		{"int + int", "{2 + 3} {3 + 2}", `5 5`},
		{"int + float", "{2 + 3.5} {3.5 + 2}", `5.5 5.5`},
		{"int + string", "{3 + '2'} {'2' + 3}", `32 23`},
		{"int + slice", "{2 + slice_var_3} {slice_var_3 + 2}", `5 5`},

		{"float + bool", "{1.5 + (1 == 1)} {(1 == 1) + 1.5}", `2.5 2.5`},
		{"float + int", "{1.5 + 3} {3 + 1.5}", `4.5 4.5`},
		{"float + float", "{1.5 + 3.5} {3.5 + 1.5}", `5 5`},
		{"float + string", "{1.5 + '2'} {'2' + 1.5}", `1.52 21.5`},
		{"float + slice", "{1.5 + slice_var_3} {slice_var_3 + 1.5}", `4.5 4.5`},

		{"string + bool", "{'A' + (1 == 1)} {(1 == 1) + 'A'}", `Atrue trueA`},
		{"string + int", "{'A' + 3} {3 + 'A'}", `A3 3A`},
		{"string + float", "{'A' + 3.5} {3.5 + 'A'}", `A3.5 3.5A`},
		{"string + string", "{'A' + '2'} {'2' + 'A'}", `A2 2A`},
		{"string + slice", "{'A' + slice_var_3} {slice_var_3 + 'A'}", `A123 123A`},

		{"slice + bool", "{slice_var_3 + (1 == 1)} {(1 == 1) + slice_var_3}", `4 4`},
		{"slice + int", "{slice_var_3 + 3} {3 + slice_var_3}", `6 6`},
		{"slice + float", "{slice_var_3 + 3.5} {3.5 + slice_var_3}", `6.5 6.5`},
		{"slice + string", "{slice_var_3 + '2'} {'2' + slice_var_3}", `1232 2123`},
		{"slice + slice", "{slice_var_3 + slice_var_3} {slice_var_3 + slice_var_3}", `6 6`},

		{"bool - bool", "{(1 == 1) - (1 >= 1)} {(1 < 1) - (1 == 1)}", `0 -1`},
		{"bool - int", "{(1 == 1) - 3} {3 - (1 == 1)}", `-2 2`},
		{"bool - float", "{(1 == 1) - 3.5} {3.5 - (1 == 1)}", `-2.5 2.5`},
		{"bool - string", "{(1 == 1) - '2'} {'foo' - (1 == 1)}", `-1 -1`},
		{"bool - slice", "{(1 == 1) - slice_var_3} {slice_var_3 - (1 == 1)}", `-2 2`},

		{"int - bool", "{(1 == 1) - 3} {3 - (1 == 1)}", `-2 2`},
		{"int - int", "{2 - 3} {3 - 2}", `-1 1`},
		{"int - float", "{2 - 3.5} {3.5 - 2}", `-1.5 1.5`},
		{"int - string", "{3 - '2'} {'foo' - 3}", `1 -3`},
		{"int - slice", "{2 - slice_var_3} {slice_var_3 - 2}", `-1 1`},

		{"float - bool", "{1.5 - (1 == 1)} {(1 == 1) - 1.5}", `0.5 -0.5`},
		{"float - int", "{1.5 - 3} {3 - 1.5}", `-1.5 1.5`},
		{"float - float", "{1.5 - 3.5} {3.5 - 1.5}", `-2 2`},
		{"float - string", "{1.5 - '2'} {'2' - 1.5}", `-0.5 0.5`},
		{"float - slice", "{1.5 - slice_var_3} {slice_var_3 - 1.5}", `-1.5 1.5`},

		{"string - bool", "{'A' - (1 == 1)} {(1 == 1) - 'AA'}", `-1 1`},
		{"string - int", "{'A' - 3} {3 - 'AA'}", `-3 3`},
		{"string - float", "{'A' - 3.5} {3.5 - 'A'}", `-3.5 3.5`},
		{"string - string", "{'A' - '22'} {'22' - 'A'}", `-22 22`},
		{"string - slice", "{'A' - slice_var_3} {slice_var_3 - 'A'}", `-3 3`},

		{"slice - bool", "{slice_var_3 - (1 == 1)} {(1 == 1) - slice_var_3}", `2 -2`},
		{"slice - int", "{slice_var_3 - 5} {5 - slice_var_3}", `-2 2`},
		{"slice - float", "{slice_var_3 - 3.5} {3.5 - slice_var_3}", `-0.5 0.5`},
		{"slice - string", "{slice_var_3 - '2'} {'2' - slice_var_3}", `1 -1`},
		{"slice - slice", "{slice_var_3 - slice_var_5} {slice_var_5 - slice_var_3}", `-2 2`},

		{"bool * bool", "{(1 == 1) * (1 >= 1)} {(1 >= 1) * (1 == 1)}", `1 1`},
		{"bool * int", "{(1 == 1) * 3} {3 * (1 < 1)}", `3 0`},
		{"bool * float", "{(1 == 1) * 3.5} {3.5 * (1 == 1)}", `3.5 3.5`},
		{"bool * string", "{(1 == 1) * '2'} {'2' * (1 == 1)}", `2 2`},
		{"bool * slice", "{(1 == 1) * slice_var_3} {slice_var_3 * (1 == 1)}", `3 3`},

		{"int * bool", "{(1 == 1) * 3} {3 * (1 == 1)}", `3 3`},
		{"int * int", "{2 * 3} {3 * 2}", `6 6`},
		{"int * float", "{2 * 3.5} {3.5 * 2}", `7 7`},
		{"int * string", "{3 * '2'} {'2' * 3}", `222 222`},
		{"int * slice", "{2 * slice_var_3} {slice_var_3 * 2}", `6 6`},

		{"float * bool", "{1.5 * (1 == 1)} {(1 == 1) * 1.5}", `1.5 1.5`},
		{"float * int", "{1.5 * 3} {3 * 1.5}", `4.5 4.5`},
		{"float * float", "{1.5 * 3.5} {3.5 * 1.5}", `5.25 5.25`},
		{"float * string", "{2.5 * '2'} {'2' * 2.5}", `22 22`},
		{"float * slice", "{1.5 * slice_var_3} {slice_var_3 * 1.5}", `4.5 4.5`},

		{"string * bool", "{'A' * (1 == 1)} {(1 == 1) * 'A'}", `A A`},
		{"string * int", "{'A' * 3} {3 * 'A'}", `AAA AAA`},
		{"string * float", "{'A' * 3.5} {3.5 * 'A'}", `AAA AAA`},
		{"string * string", "{'A' * '2'} {'2' * 'A'} {'A' * 'AA'}", `AA AA `},
		{"string * slice", "{'A' * slice_var_3} {slice_var_3 * 'A'}", `AAA AAA`},

		{"slice * bool", "{slice_var_3 * (1 == 1)} {(1 == 1) * slice_var_3}", `3 3`},
		{"slice * int", "{slice_var_3 * 3} {3 * slice_var_3}", `9 9`},
		{"slice * float", "{slice_var_3 * 3.5} {3.5 * slice_var_3}", `10.5 10.5`},
		{"slice * string", "{slice_var_3 * '2'} {'2' * slice_var_3} {'A' * slice_var_3} {slice_var_3 * 'A'}", `222 222 AAA AAA`},
		{"slice * slice", "{slice_var_3 * slice_var_3} {slice_var_3 * slice_var_3}", `9 9`},

		{"bool / bool", "{(1 == 1) / (1 >= 1)} {(1 >= 1) / (1 == 1)}", `1 1`},
		{"bool / int", "{(1 == 1) / 3} {3 / (1 < 1)}", `0 0`},
		{"bool / float", "{(1 == 1) / 3.5} {3.5 / (1 == 1)}", `0.2857142857142857 3.5`},
		{"bool / string", "{(1 == 1) / '2'} {'2' / (1 == 1)} {(1 == 1) / '2.5'}", `0 2 0.4`},
		{"bool / slice", "{(1 == 1) / slice_var_3} {slice_var_3 / (1 == 1)}", `0 3`},

		{"int / bool", "{(1 == 1) / 3} {3 / (1 == 1)}", `0 3`},
		{"int / int", "{2 / 3} {3 / 2}", `0 1`},
		{"int / float", "{2 / 3.5} {3.5 / 2}", `0.5714285714285714 1.75`},
		{"int / string", "{3 / '2'} {'2' / 3}", `1 0`},
		{"int / slice", "{2 / slice_var_3} {slice_var_3 / 2}", `0 1`},

		{"float / bool", "{1.5 / (1 == 1)} {(1 == 1) / 1.5}", `1.5 0.6666666666666666`},
		{"float / int", "{1.5 / 3} {3 / 1.5}", `0.5 2`},
		{"float / float", "{1.5 / 3.5} {3.5 / 1.5}", `0.42857142857142855 2.3333333333333335`},
		{"float / string", "{2.5 / '2'} {'2' / 2.5}", `1.25 0.8`},
		{"float / slice", "{1.5 / slice_var_3} {slice_var_3 / 1.5}", `0.5 2`},

		{"string / bool", "{'A' / (1 == 1)} {(1 == 1) / 'A'}", `0 0`},
		{"string / int", "{'A' / 3} {3 / 'A'}", `0 0`},
		{"string / float", "{'A' / 3.5} {3.5 / 'A'}", `0 0`},
		{"string / string", "{'A' / '2'} {'2' / 'A'} {'A' / 'AA'}", `0 0 0`},
		{"string / slice", "{'A' / slice_var_3} {slice_var_3 / 'A'}", `0 0`},

		{"slice / bool", "{slice_var_3 / (1 == 1)} {(1 == 1) / slice_var_3}", `3 0`},
		{"slice / int", "{slice_var_3 / 3} {3 / slice_var_3}", `1 1`},
		{"slice / float", "{slice_var_3 / 3.5} {3.5 / slice_var_3}", `0.8571428571428571 1.1666666666666667`},
		{"slice / string", "{slice_var_3 / '2'} {'2' / slice_var_3} {'A' / slice_var_3} {slice_var_3 / 'A'}", `1 0 0 0`},
		{"slice / slice", "{slice_var_3 / slice_var_3} {slice_var_3 / slice_var_3}", `1 1`},

		{"bool % bool", "{(1 == 1) % (1 >= 1)} {(1 >= 1) % (1 == 1)}", `0 0`},
		{"bool % int", "{(1 == 1) % 3} {3 % (1 < 1)}", `1 0`},
		{"bool % float", "{(1 == 1) % 3.5} {3.5 % (1 == 1)}", `1 0`},
		{"bool % string", "{(1 == 1) % '2'} {'2' % (1 == 1)} {(1 == 1) % '2.5'}", `1 0 0`},
		{"bool % slice", "{(1 == 1) % slice_var_3} {slice_var_3 % (1 == 1)}", `1 0`},

		{"int % bool", "{(1 == 1) % 3} {3 % (1 == 1)}", `1 0`},
		{"int % int", "{2 % 3} {3 % 2}", `2 1`},
		{"int % float", "{2 % 3.5} {3.5 % 2}", `2 1`},
		{"int % string", "{3 % '2'} {'2' % 3}", `1 2`},
		{"int % slice", "{2 % slice_var_3} {slice_var_3 % 2}", `2 1`},

		{"float % bool", "{1.5 % (1 == 1)} {(1 == 1) % 1.5}", `0 0`},
		{"float % int", "{1.5 % 3} {3 % 1.5}", `1 0`},
		{"float % float", "{1.5 % 3.5} {3.5 % 1.5}", `1 0`},
		{"float % string", "{2.5 % '2'} {'2' % 2.5}", `0 0`},
		{"float % slice", "{1.5 % slice_var_3} {slice_var_3 % 1.5}", `1 0`},

		{"string % bool", "{'A' % (1 == 1)} {(1 == 1) % 'A'}", `0 0`},
		{"string % int", "{'A' % 3} {3 % 'A'}", `0 0`},
		{"string % float", "{'A' % 3.5} {3.5 % 'A'}", `0 0`},
		{"string % string", "{'A' % '2'} {'2' % 'A'} {'A' % 'AA'}", `0 0 0`},
		{"string % slice", "{'A' % slice_var_3} {slice_var_3 % 'A'}", `0 0`},

		{"slice % bool", "{slice_var_3 % (1 == 1)} {(1 == 1) % slice_var_3}", `0 1`},
		{"slice % int", "{slice_var_3 % 3} {3 % slice_var_3}", `0 0`},
		{"slice % float", "{slice_var_3 % 3.5} {3.5 % slice_var_3}", `0 0`},
		{"slice % string", "{slice_var_3 % '2'} {'2' % slice_var_3} {'A' % slice_var_3} {slice_var_3 % 'A'}", `1 2 0 0`},
		{"slice % slice", "{slice_var_3 % slice_var_3} {slice_var_3 % slice_var_3}", `0 0`},

		{"bool == bool", "{(1 == 1) == (1 >= 1)} {(1 >= 1) == (1 == 1)}", `true true`},
		{"bool == int", "{(1 == 1) == 3} {0 == (1 < 1)}", `false true`},
		{"bool == float", "{(1 == 1) == 3.5} {1 == (1 == 1)}", `false true`},
		{"bool == string", "{(1 == 1) == '2'} {'true' == (1 == 1)} {(1 == 1) == '2.5'}", `false true false`},
		{"bool == slice", "{(1 == 1) == slice_var_1} {slice_var_1 == (1 == 1)}", `false false`},

		{"int == bool", "{(1 == 1) == 3} {1 == (1 == 1)}", `false true`},
		{"int == int", "{2 == 3} {3 == 3}", `false true`},
		{"int == float", "{3.5 == 3.5} {3.5 == 2}", `true false`},
		{"int == string", "{2 == '2'} {'2' == 3}", `true false`},
		{"int == slice", "{2 == slice_var_1} {slice_var_1 == 3}", `false false`},

		{"float == bool", "{1.5 == (1 == 1)} {(1 == 1) == 1.0}", `false true`},
		{"float == int", "{1.5 == 3} {1 == 1.00}", `false true`},
		{"float == float", "{1.5 == 3.5} {3.5 == 3.5}", `false true`},
		{"float == string", "{2.5 == '2'} {'2' == 2.0}", `false true`},
		{"float == slice", "{1.5 == slice_var_1} {slice_var_1 == 3.0}", `false false`},

		{"string == bool", "{'A' == (1 == 1)} {(1 == 1) == 'true'}", `false false`},
		{"string == int", "{'A' == 3} {3 == '3'}", `false true`},
		{"string == float", "{'3.5' == 3.5} {3.5 == 'A'}", `true false`},
		{"string == string", "{'A' == '2'} {'2' == 'A'} {'A' == 'A'}", `false false true`},
		{"string == slice", "{'A' == slice_var_1} {slice_var_1 == 'AAA'} {slice_var_3 == '123'}", `false false false`},

		{"slice == bool", "{slice_var_1 == (1 == 1)} {(1 == 1) == slice_var_1}", `false false`},
		{"slice == int", "{slice_var_1 == 3} {4 == slice_var_1}", `false false`},
		{"slice == float", "{slice_var_1 == 3.5} {3.0 == slice_var_1}", `false false`},
		{"slice == string", "{slice_var_1 == '3'} {'2' == slice_var_1} {'A' == slice_var_1} {slice_var_1 == '123'}", `false false false false`},
		{"slice == slice", "{slice_var_1 == slice_var_1} {slice_var_1 == slice_var_5}", `true false`},

		{"bool != bool", "{(1 == 1) != (1 >= 1)} {(1 >= 1) != (1 == 1)}", `false false`},
		{"bool != int", "{(1 == 1) != 3} {0 != (1 < 1)}", `true false`},
		{"bool != float", "{(1 == 1) != 3.5} {1 != (1 == 1)}", `true false`},
		{"bool != string", "{(1 == 1) != '2'} {'true' != (1 == 1)} {(1 == 1) != '2.5'}", `true false true`},
		{"bool != slice", "{(1 == 1) != slice_var_1} {slice_var_1 != (1 == 1)}", `true true`},

		{"int != bool", "{(1 == 1) != 3} {1 != (1 == 1)}", `true false`},
		{"int != int", "{2 != 3} {3 != 3}", `true false`},
		{"int != float", "{3.5 != 3.5} {3.5 != 2}", `false true`},
		{"int != string", "{2 != '2'} {'2' != 3}", `false true`},
		{"int != slice", "{2 != slice_var_1} {slice_var_1 != 3}", `true true`},

		{"float != bool", "{1.5 != (1 == 1)} {(1 == 1) != 1.0}", `true false`},
		{"float != int", "{1.5 != 3} {1 != 1.00}", `true false`},
		{"float != float", "{1.5 != 3.5} {3.5 != 3.5}", `true false`},
		{"float != string", "{2.5 != '2'} {'2' != 2.0}", `true false`},
		{"float != slice", "{1.5 != slice_var_1} {slice_var_1 != 3.0}", `true true`},

		{"string != bool", "{'A' != (1 == 1)} {(1 == 1) != 'true'}", `true true`},
		{"string != int", "{'A' != 3} {3 != '3'}", `true false`},
		{"string != float", "{'3.5' != 3.5} {3.5 != 'A'}", `false true`},
		{"string != string", "{'A' != '2'} {'2' != 'A'} {'A' != 'A'}", `true true false`},
		{"string != slice", "{'A' != slice_var_1} {slice_var_1 != 'AAA'} {slice_var_3 != '123'}", `true true true`},

		{"slice != bool", "{slice_var_1 != (1 == 1)} {(1 == 1) != slice_var_1}", `true true`},
		{"slice != int", "{slice_var_1 != 3} {4 != slice_var_1}", `true true`},
		{"slice != float", "{slice_var_1 != 3.5} {3.0 != slice_var_1}", `true true`},
		{"slice != string", "{slice_var_1 != '3'} {'2' != slice_var_1} {'A' != slice_var_1} {slice_var_1 != '123'}", `true true true true`},
		{"slice != slice", "{slice_var_1 != slice_var_1} {slice_var_1 != slice_var_5}", `false true`},

		{"bool < bool", "{(1 == 1) < (1 >= 1)} {(1 > 1) < (1 == 1)}", `false true`},
		{"bool < int", "{(1 == 1) < 3} {0 < (1 < 1)}", `true false`},
		{"bool < float", "{(1 == 1) < 3.5} {1 < (1 == 1)}", `true false`},
		{"bool < string", "{(1 == 1) < '2'} {'true' < (1 == 1)} {(1 == 1) < '2.5'}", `false false false`},
		{"bool < slice", "{(1 == 1) < slice_var_1} {slice_var_1 < (1 == 1)}", `false false`},

		{"int < bool", "{(1 == 1) < 3} {1 < (1 == 1)}", `true false`},
		{"int < int", "{2 < 3} {3 < 3}", `true false`},
		{"int < float", "{3.5 < 3.5} {1.5 < 2}", `false true`},
		{"int < string", "{1 < '2'} {'2' < 3}", `true true`},
		{"int < slice", "{2 < slice_var_1} {slice_var_1 < 3}", `false true`},

		{"float < bool", "{1.5 < (1 == 1)} {(1 == 1) < 1.1}", `false true`},
		{"float < int", "{1.5 < 3} {1 < 1.001}", `true true`},
		{"float < float", "{1.5 < 3.5} {3.5 < 3.5}", `true false`},
		{"float < string", "{2.5 < '2.7'} {'2' < 2.0}", `true false`},
		{"float < slice", "{1.5 < slice_var_1} {slice_var_1 < 3.0}", `false true`},

		{"string < bool", "{'A' < (1 == 1)} {(1 == 1) < 'true'}", `false false`},
		{"string < int", "{'A' < 3} {3 < 'A'}", `false true`},
		{"string < float", "{'3.5' < 3.6} {3.5 < 'A'}", `true true`},
		{"string < string", "{'A' < 'A'} {'A' < 'B'} {'B' < 'A'}", `false true false`},
		{"string < slice", "{'A' < slice_var_1} {slice_var_1 < 'AAA'} {slice_var_3 < '123'}", `false false false`},

		{"slice < bool", "{slice_var_1 < (1 == 1)} {(1 == 1) < slice_var_1}", `false false`},
		{"slice < int", "{slice_var_1 < 3} {4 < slice_var_1}", `true false`},
		{"slice < float", "{slice_var_1 < 3.5} {3.0 < slice_var_1}", `true false`},
		{"slice < string", "{slice_var_1 < '3'} {'2' < slice_var_1} {'A' < slice_var_1} {slice_var_1 < '123'}", `false false false false`},
		{"slice < slice", "{slice_var_1 < slice_var_1} {slice_var_1 < slice_var_5}", `false true`},

		{"logic not", "{!0} {!1}", `true false`},
		{"logic or", "{0 or 0} {1 or 0} {0 or 1} {1 or 1}", `false true true true`},
		{"logic and", "{0 and 0} {1 and 0} {0 and 1} {1 and 1}", `false false false true`},

		{"compare equality", "{1 == 1} {1 != 1} {1 == 2} {1 != 2}", `true false false true`},
		{"compare less", "{1 < 1} {1 <= 1} {1 < 2} {1 <= 2} {2 < 1} {2 <= 1}", `false true true true false false`},
		{"compare greater", "{1 > 1} {1 >= 1} {1 > 2} {1 >= 2} {2 > 1} {2 >= 1}", `false true false false true true`},

		{"index a slice", "{slice_var_5[0]} {slice_var_5[2]} {slice_var_5[4]}", `1 3 5`},
		{"index a slice negative in bounds", "{slice_var_5[-1]} {slice_var_5[-3]} {slice_var_5[-5]}", `5 3 1`},
		{"index a slice out of bounds positive", "{slice_var_5[5]}", ``},
		{"index a slice out of bounds negative", "{slice_var_5[-6]}", ``},

		{"slice explicit partial", "{slice_var_5[1:4]}", `234`},
		{"slice explicit full", "{slice_var_5[0:5]}", `12345`},
		{"slice explicit negative start out of bounds", "{slice_var_5[-1:4]}", ``},
		{"slice explicit negative end out of bounds", "{slice_var_5[0:-6]}", ``},
		{"slice explicit positive end out of bounds", "{slice_var_5[0:6]}", ``},
		{"slice explicit negative end", "{slice_var_5[0:-1]}", `1234`},
		{"slice implicit start", "{slice_var_5[:4]}", `1234`},
		{"slice implicit end", "{slice_var_5[1:]}", `2345`},
		{"slice implicit start and end", "{slice_var_5[:]}", `12345`},
		{"slice then index", "{slice_var_5[1:4][1]}", `3`},

		{"select from options match 1", "{int_var_1} {int_var_1 => (1 = 'second', _ = 'seconds')}", `1 second`},
		{"select from options match other", "{int_var_3} {int_var_3 => (1 = 'second', _ = 'seconds')}", `3 seconds`},
		{"select from options match 5", "5 {5 => (1 = 'second', 5 = 'Hello', _ = 'seconds')}", `5 Hello`},

		{"text", "Foo bar baz", `Foo bar baz`},
		{"text with int", "Foo {123} baz", `Foo 123 baz`},
		{"text with float", "Foo {123.999} baz", `Foo 123.999 baz`},
		{"text with string", "Foo {`Hello, World!`} baz", `Foo Hello, World! baz`},
		{"text with ident", "Foo {string_var} baz", `Foo bar baz`},
		{"text with missing ident", "Foo {missing_var} baz", `Foo  baz`},
		{"text with start expression", "{1} Foo bar baz", `1 Foo bar baz`},
		{"text with end expression", "Foo bar baz {1}", `Foo bar baz 1`},
		{"text with slice", "Foo bar baz {slice_var_3}", `Foo bar baz 123`},
		{"text with Japanese", "日本語 {slice_var_3}", `日本語 123`},
		{"text with escapes", `foo \\ \{bar} baz`, `foo \ {bar} baz`},

		{"expr with escaped quotes", `{'Hello, \'World!\''}`, `Hello, 'World!'`},
		{"basic arithmetic", "{-1 + +2.0} {(1 + 2) * 3} {1.01} {3.0 % 2}", `1 9 1.01 1`},
		{"mixing bools and ints", "{(1 == 1) * 3 / 2}", `1`},
		{"mixing bools, ints, and floats", "{(1 == 1) * 3 / 2.0}", `1.5`},
		{"float with mod", "{(1 == 1) * 3 / 2.0 % 2}", `1`},
		{"string arithmetic", "{3 * string_var + ' ' + 123 + ' ' + string_var * 3 + ' baz ' + 123}", `barbarbar 123 barbarbar baz 123`},
		{"string arithmetic Japanese", "{3 * '日本語'}", `日本語日本語日本語`},
		{"slice arithmetic", "{slice_var_3 * 2} {2 * slice_var_3}", `6 6`},
		{"index Japanese string", "{'日本語'[2]}{string_var_jp[1]}{string_var_jp[0]}", `語本日`},
		{"slice and index Japanese string", "{'日本語'[1:][1] * 3}", `語語語`},
		{"build an or list with Japanese", "{join(string_var_jp[:-1], ', ')}, or {string_var_jp[-1]}", `日, 本, or 語`},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			r := strings.NewReader(tc.input)
			node, err := p.Parse(r)
			if err != nil {
				t.Fatal(err)
			}

			value, err := i18n.Eval(node, rt, "en-GB", vars)
			if err != nil {
				t.Fatal(err)
			}

			if got := value.AsString().Value; tc.want != got {
				t.Errorf("\n\twant %v\n\tgot  %v", tc.want, got)
			}
		})
	}
}

func TestRuntimeFunctions(t *testing.T) {
	var mdrt i18n.MarkdownRuntime
	var htmlrt i18n.HTMLRuntime

	p := i18n.NewParser()
	vars := i18n.Vars{
		"slice_var_5": i18n.NewSlice([]i18n.Value{i18n.NewInt(1), i18n.NewInt(2), i18n.NewInt(3), i18n.NewInt(4), i18n.NewInt(5)}),
	}

	tt := []struct {
		name  string
		rt    i18n.Runtime
		input string
		want  string
	}{
		{"markdown join 0 args", mdrt, "{join()}", ``},
		{"markdown join 1 arg", mdrt, "{join(slice_var_5)}", `12345`},
		{"markdown join 2 args", mdrt, "{join(slice_var_5, ', ')}", `1, 2, 3, 4, 5`},
		{"html join 0 args", htmlrt, "{join()}", ``},
		{"html join 1 arg", htmlrt, "{join(slice_var_5)}", `12345`},
		{"html join 2 args", htmlrt, "{join(slice_var_5, ', ')}", `1, 2, 3, 4, 5`},

		{"markdown split 0 args", mdrt, "{split()[1]}", ``},
		{"markdown split 1 arg", mdrt, "{split('日本語')[1]}", `本`},
		{"markdown split 2 args", mdrt, "{split('日本語', '')[1]}", `本`},
		{"html split 0 args", htmlrt, "{split()[1]}", ``},
		{"html split 1 arg", htmlrt, "{split('日本語')[1]}", `本`},
		{"html split 2 args", htmlrt, "{split('日本語', '')[1]}", `本`},

		{"markdown bold 0 args", mdrt, "{bold()}", ``},
		{"markdown bold 1 arg", mdrt, "{bold('Foo')}", `**Foo**`},
		{"html bold 0 args", htmlrt, "{bold()}", ``},
		{"html bold 1 arg", htmlrt, "{bold('Foo')}", `<b>Foo</b>`},

		{"markdown italic 0 args", mdrt, "{italic()}", ``},
		{"markdown italic 1 arg", mdrt, "{italic('Foo')}", `*Foo*`},
		{"html italic 0 args", htmlrt, "{italic()}", ``},
		{"html italic 1 arg", htmlrt, "{italic('Foo')}", `<i>Foo</i>`},

		{"markdown link 0 args", mdrt, "{link()}", `[]()`},
		{"markdown link 1 arg", mdrt, "{link('Foo')}", `[Foo]()`},
		{"markdown link 2 args", mdrt, "{link('Foo', '/bar')}", `[Foo](/bar)`},
		{"markdown link 3 args", mdrt, "{link('Foo', '/bar', '_blank')}", `[Foo](/bar)`},
		{"html link 0 args", htmlrt, "{link()}", `<a href=""></a>`},
		{"html link 1 arg", htmlrt, "{link('Foo')}", `<a href="">Foo</a>`},
		{"html link 2 args", htmlrt, "{link('Foo', '/bar')}", `<a href="/bar">Foo</a>`},
		{"html link 3 args", htmlrt, "{link('Foo', '/bar', '_blank')}", `<a href="/bar" target="_blank">Foo</a>`},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			r := strings.NewReader(tc.input)
			node, err := p.Parse(r)
			if err != nil {
				t.Fatal(err)
			}

			value, err := i18n.Eval(node, tc.rt, "en-GB", vars)
			if err != nil {
				t.Fatal(err)
			}

			if got := value.AsString().Value; tc.want != got {
				t.Errorf("\n\twant %v\n\tgot  %v", tc.want, got)
			}
		})
	}
}
