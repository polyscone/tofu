package i18n

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"maps"
	"path"
	"strings"
	"sync"
)

var ErrNotFound = errors.New("message not found")

var (
	FallbackLocale = "en-GB"
	ErrorLocale    = FallbackLocale
)

var (
	resourcesMu     sync.Mutex
	resources       = make(map[string]*Resource)
	resourceLocales []string
)

func closestLocale(candidates []string) (string, bool) {
	for _, candidate := range candidates {
		locales := strings.Split(candidate, ",")

		for _, locale := range locales {
			if _, ok := resources[locale]; ok {
				return locale, true
			}

			locale, _, _ = strings.Cut(locale, "-")

			if _, ok := resources[locale]; ok {
				return locale, true
			}

			prefix := locale + "-"
			for _, rl := range resourceLocales {
				if strings.HasPrefix(rl, prefix) {
					return rl, true
				}
			}
		}
	}

	return "", false
}

func ClosestLocale(candidates []string) (string, bool) {
	if locale, ok := closestLocale(candidates); ok {
		return locale, true
	}

	return FallbackLocale, false
}

func LoadJSON(locale string, r io.Reader) error {
	resourcesMu.Lock()
	defer resourcesMu.Unlock()

	rsc := resources[locale]
	if rsc == nil {
		rsc = NewResource()

		resources[locale] = rsc
	}

	if err := rsc.LoadJSON(r); err != nil {
		return fmt.Errorf("locale %v: %w", locale, err)
	}

	resourceLocales = append(resourceLocales, locale)

	return nil
}

func LoadJSONFiles(locales fs.FS) error {
	return fs.WalkDir(locales, ".", func(p string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		ext := path.Ext(p)
		if ext != ".json" {
			return nil
		}

		f, err := locales.Open(p)
		if err != nil {
			return err
		}
		defer f.Close()

		locale := strings.TrimSuffix(path.Base(p), ext)

		return LoadJSON(locale, f)
	})
}

func T(rt Runtime, locale string, message Message) (Value, error) {
	if message.Key == "" {
		return stringEmpty, nil
	}

	rsc := resources[locale]
	if rsc == nil {
		return NewString(message.String()), ErrNotFound
	}

	return rsc.T(rt, locale, message)
}

type Message struct {
	Key  string
	Vars Vars
}

func M(id string, pairs ...any) Message {
	context, key, ok := strings.Cut(id, ":")
	if ok {
		pairs = append(pairs, "context", context)
	} else {
		context, key = key, context
	}

	vars, err := NewVars(pairs)
	if err != nil {
		panic(err)
	}

	return Message{
		Key:  key,
		Vars: vars,
	}
}

func (m Message) String() string {
	if m.Vars == nil {
		return m.Key
	}

	return fmt.Sprintf("%v: %+v", m.Key, m.Vars)
}

func (m Message) Error() string {
	s, _ := T(DefaultMarkdownRuntime, ErrorLocale, m)

	return s.AsString().Value
}

func (m Message) WithContext(context string) Message {
	return Message{
		Key:  m.Key,
		Vars: maps.Clone(m.Vars),
	}
}

func (m Message) Is(target error) bool {
	if target, ok := target.(Message); ok {
		return m.Key == target.Key
	}

	return false
}

type Translation struct {
	Condition Node
	Value     Node
	Group     []Translation
}

type Translations map[string]Translation

type Resource struct {
	translations Translations
	parser       *Parser
}

func NewResource() *Resource {
	return &Resource{
		translations: make(Translations),
		parser:       NewParser(),
	}
}

func (rsc *Resource) LoadJSON(r io.Reader) error {
	var data map[string]any
	if err := json.NewDecoder(r).Decode(&data); err != nil {
		return fmt.Errorf("decode translation JSON: %w", err)
	}

	for key, value := range data {
		flatten(data, key, value)
	}

	for key, value := range data {
		if _, ok := value.(map[string]any); ok {
			continue
		}

		t, err := rsc.loadTranslation(value)
		if err != nil {
			return fmt.Errorf("%v: %w", key, err)
		}

		rsc.translations[key] = t
	}

	return nil
}

func flatten(m map[string]any, key string, value any) {
	switch value := value.(type) {
	case map[string]any:
		for suffix, val := range value {
			full := key + "." + suffix

			flatten(m, full, val)
		}

	case []any, string:
		m[key] = value
	}
}

func (rsc *Resource) loadTranslation(value any) (Translation, error) {
	var t Translation
	switch value := value.(type) {
	case []any:
		switch len(value) {
		case 0:
			return t, nil

		case 1:
			s, ok := value[0].(string)
			if !ok {
				break
			}

			val, err := rsc.parser.Parse(strings.NewReader(s))
			if err != nil {
				return t, err
			}

			t.Value = val

		default:
			s0, ok0 := value[0].(string)
			if !ok0 {
				break
			}

			elements := value
			if strings.HasPrefix(s0, "if:") {
				s0 = strings.TrimPrefix(s0, "if:")
				if !strings.HasPrefix(s0, "{") {
					s0 = "{" + s0
				}
				if !strings.HasSuffix(s0, "}") {
					s0 += "}"
				}

				node, err := rsc.parser.Parse(strings.NewReader(s0))
				if err != nil {
					return t, err
				}

				var cond Node
				if len(node.Fragments) > 0 {
					cond = node.Fragments[0]
				}

				t.Condition = cond

				elements = value[1:]
			}

			var strs []string
			for _, v := range elements {
				s, ok := v.(string)
				if ok {
					strs = append(strs, s)
				}
			}
			if len(elements) == len(strs) {
				multiline := strings.Join(strs, "\n")
				val, err := rsc.parser.Parse(strings.NewReader(multiline))
				if err != nil {
					return t, err
				}

				t.Value = val
			}
		}

		if t.Value != nil {
			return t, nil
		}

		t.Group = make([]Translation, len(value))
		for i, candidate := range value {
			if i == 0 && t.Condition != nil {
				continue
			}

			translation, err := rsc.loadTranslation(candidate)
			if err != nil {
				return t, err
			}

			t.Group[i] = translation
		}

	case string:
		val, err := rsc.parser.Parse(strings.NewReader(value))
		if err != nil {
			return t, err
		}

		t.Value = val

	default:
		return t, fmt.Errorf("cannot load %T", value)
	}

	return t, nil
}

func (rsc *Resource) choose(t Translation, rt Runtime, locale string, vars Vars) (Node, error) {
	if t.Condition != nil {
		res, err := Eval(t.Condition, rt, locale, vars)
		if err != nil {
			return nil, fmt.Errorf("eval condition: %w", err)
		}
		if !res.AsBool().Value {
			return nil, nil
		}
	}

	if t.Value != nil {
		return t.Value, nil
	}

	for _, candidate := range t.Group {
		node, err := rsc.choose(candidate, rt, locale, vars)
		if err != nil {
			return nil, err
		}
		if node != nil {
			return node, nil
		}
	}

	return nil, nil
}

func (rsc *Resource) T(rt Runtime, locale string, message Message) (Value, error) {
	if message.Key == "" {
		return stringEmpty, nil
	}

	node, err := rsc.choose(rsc.translations[message.Key], rt, locale, message.Vars)
	if err != nil {
		return NewString(message.String()), fmt.Errorf("choose: %w", err)
	}
	if node == nil {
		return NewString(message.String()), ErrNotFound
	}

	return Eval(node, rt, locale, message.Vars)
}
