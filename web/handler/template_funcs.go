package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/polyscone/tofu/pkg/human"
	"github.com/polyscone/tofu/web/httputil"
)

func NewTemplateFuncs(custom template.FuncMap) template.FuncMap {
	funcs := template.FuncMap{
		"Add":                TmplAdd,
		"Sub":                TmplSub,
		"Mul":                TmplMul,
		"Div":                TmplDiv,
		"Mod":                TmplMod,
		"Neg":                TmplNeg,
		"Max":                TmplMax,
		"Min":                TmplMin,
		"Addf":               TmplAddf,
		"Subf":               TmplSubf,
		"Mulf":               TmplMulf,
		"Divf":               TmplDivf,
		"Negf":               TmplNegf,
		"Maxf":               TmplMaxf,
		"Minf":               TmplMinf,
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

func toInts(values []any) (int, []int, error) {
	if len(values) < 2 {
		return 0, nil, errors.New("at least two operands required")
	}

	nums := make([]int, len(values))
	for i, v := range values {
		switch v := v.(type) {
		case int:
			nums[i] = v

		case float64:
			nums[i] = int(v)

		default:
			return 0, nil, errors.New("expected int or float64")
		}
	}

	return nums[0], nums[1:], nil
}

func TmplAdd(values ...any) (int, error) {
	acc, nums, err := toInts(values)
	if err != nil {
		return 0, fmt.Errorf("Add: %v", err)
	}

	for _, num := range nums {
		acc += num
	}

	return acc, nil
}

func TmplSub(values ...any) (int, error) {
	acc, nums, err := toInts(values)
	if err != nil {
		return 0, fmt.Errorf("Sub: %v", err)
	}

	for _, num := range nums {
		acc -= num
	}

	return acc, nil
}

func TmplMul(values ...any) (int, error) {
	acc, nums, err := toInts(values)
	if err != nil {
		return 0, fmt.Errorf("Mul: %v", err)
	}

	for _, num := range nums {
		acc *= num
	}

	return acc, nil
}

func TmplDiv(values ...any) (int, error) {
	acc, nums, err := toInts(values)
	if err != nil {
		return 0, fmt.Errorf("Div: %v", err)
	}

	for _, num := range nums {
		acc /= num
	}

	return acc, nil
}

func TmplMod(values ...any) (int, error) {
	acc, nums, err := toInts(values)
	if err != nil {
		return 0, fmt.Errorf("Mod: %v", err)
	}

	for _, num := range nums {
		acc %= num
	}

	return acc, nil
}

func TmplNeg(x any) (int, error) {
	switch x := x.(type) {
	case int:
		return -x, nil

	case float64:
		return -int(x), nil

	default:
		return 0, errors.New("Neg: expected int or float64")
	}
}

func TmplMax(values ...any) (int, error) {
	first, rest, err := toInts(values)
	if err != nil {
		return 0, fmt.Errorf("Max: %v", err)
	}

	return slices.Max(slices.Concat([]int{first}, rest)), nil
}

func TmplMin(values ...any) (int, error) {
	first, rest, err := toInts(values)
	if err != nil {
		return 0, fmt.Errorf("Min: %v", err)
	}

	return slices.Min(slices.Concat([]int{first}, rest)), nil
}

func toFloat64s(values []any) (float64, []float64, error) {
	if len(values) < 2 {
		return 0, nil, errors.New("at least two operands required")
	}

	nums := make([]float64, len(values))
	for i, v := range values {
		switch v := v.(type) {
		case float64:
			nums[i] = v

		case int:
			nums[i] = float64(v)

		default:
			return 0, nil, errors.New("expected int or float64")
		}
	}

	return nums[0], nums[1:], nil
}

func TmplAddf(values ...any) (float64, error) {
	acc, nums, err := toFloat64s(values)
	if err != nil {
		return 0, fmt.Errorf("Addf: %v", err)
	}

	for _, num := range nums {
		acc += num
	}

	return acc, nil
}

func TmplSubf(values ...any) (float64, error) {
	acc, nums, err := toFloat64s(values)
	if err != nil {
		return 0, fmt.Errorf("Subf: %v", err)
	}

	for _, num := range nums {
		acc -= num
	}

	return acc, nil
}

func TmplMulf(values ...any) (float64, error) {
	acc, nums, err := toFloat64s(values)
	if err != nil {
		return 0, fmt.Errorf("Mulf: %v", err)
	}

	for _, num := range nums {
		acc *= num
	}

	return acc, nil
}

func TmplDivf(values ...any) (float64, error) {
	acc, nums, err := toFloat64s(values)
	if err != nil {
		return 0, fmt.Errorf("Divf: %v", err)
	}

	for _, num := range nums {
		acc /= num
	}

	return acc, nil
}

func TmplNegf(x any) (float64, error) {
	switch x := x.(type) {
	case float64:
		return -x, nil

	case int:
		return -float64(x), nil

	default:
		return 0, errors.New("Negf: expected int or float64")
	}
}

func TmplMaxf(values ...any) (float64, error) {
	first, rest, err := toFloat64s(values)
	if err != nil {
		return 0, fmt.Errorf("Maxf: %v", err)
	}

	return slices.Max(slices.Concat([]float64{first}, rest)), nil
}

func TmplMinf(values ...any) (float64, error) {
	first, rest, err := toFloat64s(values)
	if err != nil {
		return 0, fmt.Errorf("Minf: %v", err)
	}

	return slices.Min(slices.Concat([]float64{first}, rest)), nil
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
