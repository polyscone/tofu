package human

import "strings"

func List(strs []string) string {
	switch n := len(strs); n {
	case 0:
		return ""

	case 1:
		return strs[0]

	case 2:
		return strs[0] + " and " + strs[1]

	default:
		first, last := strs[:n-1], strs[n-1]

		return strings.Join(first, ", ") + " and " + last
	}
}
