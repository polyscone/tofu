package handler

import (
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/http/router"
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

func tmplStatusText(code int) string {
	return strings.ReplaceAll(http.StatusText(code), "z", "s")
}

type tmplPathFunc func(name string, paramArgPairs ...any) template.URL

func tmplPath(mux *router.ServeMux) tmplPathFunc {
	return func(name string, paramArgPairs ...any) template.URL {
		return template.URL(mux.Path(name, paramArgPairs...))
	}
}

func tmplQueryReplace(q url.Values, pairs ...any) (url.Values, error) {
	if len(pairs)%2 == 1 {
		return nil, errors.Tracef("QueryString expects pairs of key value replacements")
	}

	u, err := url.Parse("?" + q.Encode())
	if err != nil {
		return nil, errors.Tracef(err)
	}

	q = u.Query()
	for i := 0; i < len(pairs); i += 2 {
		key := fmt.Sprintf("%v", pairs[i])
		value := pairs[i+1]

		if value == nil {
			q.Del(key)

			continue
		}

		q.Set(key, fmt.Sprintf("%v", value))
	}

	return q, nil
}

func tmplQueryURL(q url.Values) template.URL {
	value := q.Encode()

	if value == "" {
		return ""
	}

	if !strings.HasPrefix(value, "?") {
		value = "?" + value
	}

	return template.URL(value)
}

func tmplQueryString(q url.Values, pairs ...any) (template.URL, error) {
	q, err := tmplQueryReplace(q, pairs...)
	if err != nil {
		return "", errors.Tracef(err)
	}

	for key, values := range q {
		var keep bool
		for _, value := range values {
			if value != "" {
				keep = true

				break
			}
		}

		if !keep {
			q.Del(key)
		}
	}

	return tmplQueryURL(q), nil
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

func tmplHasPrefix(value, prefix any) bool {
	v := fmt.Sprintf("%v", value)
	p := fmt.Sprintf("%v", prefix)

	return strings.HasPrefix(v, p)
}

func tmplHasSuffix(value, suffix any) bool {
	v := fmt.Sprintf("%v", value)
	s := fmt.Sprintf("%v", suffix)

	return strings.HasSuffix(v, s)
}

type tmplHasPathPrefixFunc func(value any, name string, paramArgPairs ...any) bool

func tmplHasPathPrefix(mux *router.ServeMux) tmplHasPathPrefixFunc {
	return func(value any, name string, paramArgPairs ...any) bool {
		v := fmt.Sprintf("%v", value)
		p := mux.Path(name, paramArgPairs...)
		p = strings.TrimSuffix(p, "/")

		return v == p || strings.HasPrefix(v, p+"/")
	}
}

func tmplHasString(haystack []string, value any) bool {
	needle := fmt.Sprintf("%v", value)

	for _, value := range haystack {
		if value == needle {
			return true
		}
	}

	return false
}

func tmplToStrings(value any) ([]string, error) {
	switch value := value.(type) {
	case []int:
		slice := make([]string, len(value))
		for i, value := range value {
			slice[i] = strconv.Itoa(value)
		}

		return slice, nil

	case []string:
		return value, nil

	default:
		return nil, errors.Tracef("unsupported value type %T", value)
	}
}

func tmplUnescapeHTML(s string) template.HTML {
	return template.HTML(s)
}
