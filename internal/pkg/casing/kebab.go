package casing

import (
	"regexp"
	"strings"
)

var (
	reFirstUpper = regexp.MustCompile("(.)([A-Z][a-z]+)")
	reAllUppers  = regexp.MustCompile("([a-z0-9])([A-Z])")
)

func ToKebab(str string) string {
	kebab := reFirstUpper.ReplaceAllString(str, "${1}-${2}")
	kebab = reAllUppers.ReplaceAllString(kebab, "${1}-${2}")

	return strings.ToLower(kebab)
}
