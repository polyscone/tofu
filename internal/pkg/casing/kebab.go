package casing

import (
	"regexp"
	"strings"
)

var (
	matchFirstUpper = regexp.MustCompile("(.)([A-Z][a-z]+)")
	matchAllUppers  = regexp.MustCompile("([a-z0-9])([A-Z])")
)

func ToKebab(str string) string {
	kebab := matchFirstUpper.ReplaceAllString(str, "${1}-${2}")
	kebab = matchAllUppers.ReplaceAllString(kebab, "${1}-${2}")

	return strings.ToLower(kebab)
}
