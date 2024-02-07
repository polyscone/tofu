package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"math"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/polyscone/tofu/internal/adapter/web/httputil"
	"github.com/polyscone/tofu/internal/pkg/human"
)

func NewTemplateFuncs(custom template.FuncMap) template.FuncMap {
	funcs := template.FuncMap{
		"Add":                TmplAdd,
		"Sub":                TmplSub,
		"Mul":                TmplMul,
		"Div":                TmplDiv,
		"Mod":                TmplMod,
		"Addf":               TmplAddf,
		"Subf":               TmplSubf,
		"Mulf":               TmplMulf,
		"Divf":               TmplDivf,
		"Pi":                 TmplPi,
		"Cos":                TmplCos,
		"Sin":                TmplSin,
		"Ints":               TmplInts,
		"StatusText":         TmplStatusText,
		"QueryString":        TmplQueryString,
		"QueryReplace":       TmplQueryReplace,
		"TimeSince":          TmplTimeSince,
		"FormatTime":         TmplFormatTime,
		"FormatDuration":     TmplFormatDuration,
		"FormatDurationStat": TmplFormatDurationStat,
		"FormatSizeSI":       TmplFormatSizeSI,
		"FormatSizeIEC":      TmplFormatSizeIEC,
		"HasPrefix":          TmplHasPrefix,
		"HasSuffix":          TmplHasSuffix,
		"HasString":          TmplHasString,
		"ToStrings":          TmplToStrings,
		"Join":               TmplJoin,
		"ReplaceAll":         TmplReplaceAll,
		"MarshalJSON":        TmplMarshalJSON,
		"UnescapeHTML":       TmplUnescapeHTML,
		"UnescapeJS":         TmplUnescapeJS,
		"Slice":              TmplSlice,
		"SliceContains":      TmplSliceContains,
		"Map":                TmplMap,
	}

	for key, value := range custom {
		funcs[key] = value
	}

	return funcs
}

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

func TmplAddf(a, b float64) float64 {
	return a + b
}

func TmplSubf(a, b float64) float64 {
	return a - b
}

func TmplMulf(a, b float64) float64 {
	return a * b
}

func TmplDivf(a, b float64) float64 {
	return a / b
}

func TmplPi() float64 {
	return math.Pi
}

func TmplCos(x float64) float64 {
	return math.Cos(x)
}

func TmplSin(x float64) float64 {
	return math.Sin(x)
}

func TmplInts(start, end int) []int {
	n := end - start
	ints := make([]int, n)
	for i := range n {
		ints[i] = start + i
	}

	return ints
}

func TmplStatusText(code int) string {
	if code == httputil.StatusClientClosedRequest {
		return "Client Closed Request"
	}

	return strings.ReplaceAll(http.StatusText(code), "z", "s")
}

func TmplQueryString(q url.Values) template.URL {
	value := q.Encode()

	if value == "" {
		return ""
	}

	if !strings.HasPrefix(value, "?") {
		value = "?" + value
	}

	return template.URL(value)
}

func TmplQueryReplace(q url.Values, pairs ...any) (template.URL, error) {
	if len(pairs) == 0 {
		return TmplQueryString(q), nil
	}
	if len(pairs)%2 == 1 {
		return "", errors.New("QueryReplace: want pairs of key value replacements")
	}

	// We re-parse the encoded query values here so we can make a copy and not
	// alter the values that were passed in
	u, err := url.Parse("?" + q.Encode())
	if err != nil {
		return "", fmt.Errorf("QueryReplace: parse URL: %w", err)
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

	// Filter out all keys that have nothing but empty values
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

	return TmplQueryString(q), nil
}

func TmplTimeSince(t time.Time) time.Duration {
	return time.Since(t)
}

func TmplFormatTime(t time.Time, format string) string {
	switch format {
	case "Kitchen":
		return t.Format(time.Kitchen)

	case "DateTime":
		return t.Format(time.DateTime)

	case "DateOnly":
		return t.Format(time.DateOnly)

	case "TimeOnly":
		return t.Format(time.TimeOnly)

	case "RFC1123":
		return t.Format(time.RFC1123)

	case "RFC3339":
		return t.Format(time.RFC3339)
	}

	return t.Format(format)
}

func TmplFormatDuration(d time.Duration) string {
	return human.Duration(d)
}

func TmplFormatDurationStat(d time.Duration) string {
	return human.DurationStat(d)
}

func TmplFormatSizeSI(bytes uint64) string {
	return human.SizeSI(bytes)
}

func TmplFormatSizeIEC(bytes uint64) string {
	return human.SizeIEC(bytes)
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

func TmplReplaceAll(value any, old, new string) string {
	str := fmt.Sprintf("%v", value)

	return strings.ReplaceAll(str, old, new)
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

func TmplUnescapeJS(s string) template.JS {
	return template.JS(s)
}

func TmplSlice(elements ...any) []any {
	return elements
}

func TmplSliceContains(haystack []any, needle any) bool {
	return slices.Contains(haystack, needle)
}

func TmplMap(pairs ...any) (map[string]any, error) {
	if len(pairs)%2 == 1 {
		return nil, errors.New("Map: want key value pairs")
	}

	m := make(map[string]any, len(pairs)/2)
	for i := 0; i < len(pairs); i += 2 {
		key := fmt.Sprintf("%v", pairs[i])
		value := pairs[i+1]

		m[key] = value
	}

	return m, nil
}
