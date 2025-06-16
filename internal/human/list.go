package human

import "strings"

func List(strs []string, sep, conjunction string) string {
	out := make([]string, len(strs))
	for i, str := range strs {
		switch str {
		case "\t":
			out[i] = "<Tab>"

		case "\r":
			out[i] = "<CR>"

		case "\n":
			out[i] = "<LF>"

		default:
			out[i] = str
		}
	}

	switch n := len(out); n {
	case 0:
		return ""

	case 1:
		return out[0]

	case 2:
		return out[0] + conjunction + out[1]

	default:
		first, last := out[:n-1], out[n-1]

		return strings.Join(first, sep) + "," + conjunction + last
	}
}

func AndList(strs []string) string {
	return List(strs, ", ", " and ")
}

func AndListString(str string) string {
	strs := strings.Split(str, "")

	return AndList(strs)
}

func OrList(strs []string) string {
	return List(strs, ", ", " or ")
}

func OrListString(str string) string {
	strs := strings.Split(str, "")

	return OrList(strs)
}
