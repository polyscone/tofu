package handler

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/polyscone/tofu/internal/pkg/http/router"
)

func TmplAdd(a, b int) int {
	return a + b
}

func TmplSub(a, b int) int {
	return a - b
}

func TmplMul(a, b int) int {
	return a * b
}

func TmplDiv(a, b int) int {
	return a / b
}

func TmplMod(a, b int) int {
	return a % b
}

func TmplInts(start, end int) []int {
	n := end - start
	ints := make([]int, n)
	for i := 0; i < n; i++ {
		ints[i] = start + i
	}

	return ints
}

func TmplStatusText(code int) string {
	return strings.ReplaceAll(http.StatusText(code), "z", "s")
}

type tmplPathFunc func(name string, paramArgPairs ...any) template.URL

func TmplPath(mux *router.ServeMux) tmplPathFunc {
	return func(name string, paramArgPairs ...any) template.URL {
		return template.URL(mux.Path(name, paramArgPairs...))
	}
}

func TmplQueryReplace(q url.Values, pairs ...any) (url.Values, error) {
	if len(pairs)%2 == 1 {
		return nil, fmt.Errorf("QueryReplace: want pairs of key value replacements")
	}

	u, err := url.Parse("?" + q.Encode())
	if err != nil {
		return nil, fmt.Errorf("QueryReplace: parse URL: %w", err)
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

func TmplQueryURL(q url.Values) template.URL {
	value := q.Encode()

	if value == "" {
		return ""
	}

	if !strings.HasPrefix(value, "?") {
		value = "?" + value
	}

	return template.URL(value)
}

func TmplQueryString(q url.Values, pairs ...any) (template.URL, error) {
	q, err := TmplQueryReplace(q, pairs...)
	if err != nil {
		return "", fmt.Errorf("QueryString: %w", err)
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

	return TmplQueryURL(q), nil
}

func TmplFormatTime(t time.Time, format string) string {
	switch format {
	case "DateTime":
		return t.Format(time.DateTime)

	case "RFC3339":
		return t.Format(time.RFC3339)
	}

	return t.Format(format)
}

func TmplHasPrefix(value, prefix any) bool {
	v := fmt.Sprintf("%v", value)
	p := fmt.Sprintf("%v", prefix)

	return strings.HasPrefix(v, p)
}

func TmplHasSuffix(value, suffix any) bool {
	v := fmt.Sprintf("%v", value)
	s := fmt.Sprintf("%v", suffix)

	return strings.HasSuffix(v, s)
}

type tmplHasPathPrefixFunc func(value any, name string, paramArgPairs ...any) bool

func TmplHasPathPrefix(mux *router.ServeMux) tmplHasPathPrefixFunc {
	return func(value any, name string, paramArgPairs ...any) bool {
		v := fmt.Sprintf("%v", value)
		p := mux.Path(name, paramArgPairs...)
		p = strings.TrimSuffix(p, "/")

		return v == p || strings.HasPrefix(v, p+"/")
	}
}

func TmplHasString(haystack []string, value any) bool {
	needle := fmt.Sprintf("%v", value)

	for _, value := range haystack {
		if value == needle {
			return true
		}
	}

	return false
}

func TmplToStrings(value any) ([]string, error) {
	switch value := value.(type) {
	case nil:
		return nil, nil

	case []int:
		slice := make([]string, len(value))
		for i, value := range value {
			slice[i] = strconv.Itoa(value)
		}

		return slice, nil

	case []string:
		return value, nil

	default:
		return nil, fmt.Errorf("unsupported value type %T", value)
	}
}

func TmplJoin(strs []string, sep string) string {
	return strings.Join(strs, sep)
}

func TmplMarshalJSON(value any) (string, error) {
	b, err := json.Marshal(value)
	if err != nil {
		return "", fmt.Errorf("template marshal JSON: %w", err)
	}

	return string(b), nil
}

func TmplUnescapeHTML(s string) template.HTML {
	return template.HTML(s)
}
