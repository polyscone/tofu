package human

import "strings"

func List(strs []string, sep, conjunction string) string {
	for i, str := range strs {
		switch str {
		case "\r":
			strs[i] = "<CR>"

		case "\n":
			strs[i] = "<LF>"
		}
	}

	switch n := len(strs); n {
	case 0:
		return ""

	case 1:
		return strs[0]

	case 2:
		return strs[0] + conjunction + strs[1]

	default:
		first, last := strs[:n-1], strs[n-1]

		return strings.Join(first, sep) + conjunction + last
	}
}

func AndList(strs []string) string {
	return List(strs, ", ", " and ")
}

func OrList(strs []string) string {
	return List(strs, ", ", " or ")
}
