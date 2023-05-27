package handler

import (
	"fmt"
	"html/template"
	"net/url"
	"strings"
	"time"

	"github.com/polyscone/tofu/internal/pkg/errors"
)

func tmplAdd(a, b int) int {
	return a + b
}

func tmplSub(a, b int) int {
	return a - b
}

func tmplMul(a, b int) int {
	return a * b
}

func tmplDiv(a, b int) int {
	return a / b
}

func tmplMod(a, b int) int {
	return a % b
}

func tmplInts(start, end int) []int {
	n := end - start
	ints := make([]int, n)
	for i := 0; i < n; i++ {
		ints[i] = start + i
	}

	return ints
}

func tmplHTML(s string) template.HTML {
	return template.HTML(s)
}

func tmplHTMLAttr(s string) template.HTMLAttr {
	return template.HTMLAttr(s)
}

func tmplURL(s string) template.URL {
	return template.URL(s)
}

func tmplQueryString(q url.Values) string {
	value := q.Encode()

	if value == "" {
		return ""
	}

	if !strings.HasPrefix(value, "?") {
		value = "?" + value
	}

	return value
}

func tmplQueryReplace(q url.Values, pairs ...any) (string, error) {
	if len(pairs)%2 == 1 {
		return "", errors.Tracef("QueryReplace expects pairs of key value replacements")
	}

	u, err := url.Parse("?" + q.Encode())
	if err != nil {
		return "", errors.Tracef(err)
	}

	qq := u.Query()
	for i := 0; i < len(pairs); i += 2 {
		key := fmt.Sprintf("%v", pairs[i])
		value := pairs[i+1]

		if value == nil {
			qq.Del(key)

			continue
		}

		qq.Set(key, fmt.Sprintf("%v", value))
	}

	return tmplQueryString(qq), nil
}

func tmplFormatTime(t time.Time, format string) string {
	switch format {
	case "DateTime":
		return t.Format(time.DateTime)

	case "RFC3339":
		return t.Format(time.RFC3339)
	}

	return t.Format(format)
}
