package human

import (
	"slices"
	"strings"
)

func List(strs []string, sep, conjunction string) string {
	slices.Sort(strs)

	strs = slices.Compact(strs)

	for i, str := range strs {
		switch str {
		case "\t":
			strs[i] = "<Tab>"

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

		return strings.Join(first, sep) + ", " + conjunction + last
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
