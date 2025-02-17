package i18n

import (
	"fmt"
	"html/template"
	"math"
	"strings"
	"time"
	"unicode/utf8"
)

type MarkdownRuntime struct{}

func (r MarkdownRuntime) Kind() string {
	return "markdown"
}

func (r MarkdownRuntime) Len(value Value) Int {
	slice := value.AsSlice()

	return NewInt(int64(len(slice)))
}

func (r MarkdownRuntime) Join(s, sep Value) String {
	slice := s.AsSlice()
	strs := make([]string, len(slice))
	for i, value := range slice {
		strs[i] = value.AsString().Value
	}

	return NewString(strings.Join(strs, sep.AsString().Value))
}

func (r MarkdownRuntime) Split(s, sep Value) Slice {
	str := s.AsString().Value
	separator := sep.AsString().Value
	strs := strings.Split(str, separator)

	values := make([]Value, len(strs))
	for i, str := range strs {
		values[i] = NewString(str)
	}

	return NewSlice(values)
}

func (r MarkdownRuntime) Bold(value Value) RawString {
	s := value.AsString().Value
	if s == "" {
		return rawStringEmpty
	}

	return NewRawString("**" + s + "**")
}

func (r MarkdownRuntime) Italic(value Value) RawString {
	s := value.AsString().Value
	if s == "" {
		return rawStringEmpty
	}

	return NewRawString("*" + s + "*")
}

func (r MarkdownRuntime) Link(label, href, target Value) RawString {
	_label := template.HTMLEscapeString(label.AsString().Value)
	_href := template.HTMLEscapeString(href.AsString().Value)
	link := fmt.Sprintf("[%v](%v)", _label, _href)

	return NewRawString(link)
}

func (r MarkdownRuntime) PadLeft(value, length, padding Value) String {
	s := value.AsString().Value
	n := int(length.AsInt().Value)
	padlen := max(0, n-utf8.RuneCountInString(s))
	if padlen <= 0 {
		return value.AsString()
	}

	var sb strings.Builder
	padrunes := []rune(padding.AsString().Value)
	for i := 0; i < padlen; i++ {
		sb.WriteRune(padrunes[i%len(padrunes)])
	}

	return NewString(sb.String() + s)
}

func (r MarkdownRuntime) PadRight(value, length, padding Value) String {
	s := value.AsString().Value
	n := int(length.AsInt().Value)
	padlen := max(0, n-utf8.RuneCountInString(s))
	if padlen <= 0 {
		return value.AsString()
	}

	var sb strings.Builder
	padrunes := []rune(padding.AsString().Value)
	for i := 0; i < padlen; i++ {
		sb.WriteRune(padrunes[i%len(padrunes)])
	}

	return NewString(s + sb.String())
}

func (r MarkdownRuntime) TrimLeft(value, trim Value) String {
	_value := value.AsString().Value
	_trim := trim.AsString().Value

	return NewString(strings.TrimPrefix(_value, _trim))
}

func (r MarkdownRuntime) TrimRight(value, trim Value) String {
	_value := value.AsString().Value
	_trim := trim.AsString().Value

	return NewString(strings.TrimSuffix(_value, _trim))
}

func (r MarkdownRuntime) Integer(value Value) Int {
	integer, _ := math.Modf(value.AsFloat().Value)

	return NewInt(int64(integer))
}

func (r MarkdownRuntime) Fraction(value, roundingUnit Value) Float {
	_, frac := math.Modf(value.AsFloat().Value)
	if unit := roundingUnit.AsFloat().Value; unit > 0 {
		frac = math.Round(frac/unit) * unit
	}

	return NewFloat(frac)
}

func (r MarkdownRuntime) T(key Value, locale string, value, opt Value) String {
	vars := Vars{
		"value": value,
		"opt":   opt,
	}
	switch value := value.(type) {
	case Time:
		t := value.Value
		zoneName := t.Location().String()
		zoneNameShort, _ := t.Zone()
		zoneOffset := t.Format("-07:00")

		vars["year"] = NewInt(int64(t.Year()))
		vars["month"] = NewInt(int64(t.Month()))
		vars["day"] = NewInt(int64(t.Day()))
		vars["weekday"] = NewString(strings.ToLower(t.Weekday().String()))
		vars["hour"] = NewInt(int64(t.Hour()))
		vars["minute"] = NewInt(int64(t.Minute()))
		vars["second"] = NewInt(int64(t.Second()))
		vars["zone_name"] = NewString(zoneName)
		vars["zone_name_short"] = NewString(zoneNameShort)
		vars["zone_offset"] = NewString(zoneOffset)

	case Int, Duration:
		var d time.Duration
		switch value := value.(type) {
		case Int:
			d = time.Duration(value.Value)

		case Duration:
			d = value.Value
		}

		vars["years"] = NewInt(int64(d.Hours() / 24 / 365))
		vars["days"] = NewInt(int64(math.Mod(d.Hours()/24, 365)))
		vars["hours"] = NewInt(int64(math.Mod(d.Hours(), 24)))
		vars["minutes"] = NewInt(int64(math.Mod(d.Minutes(), 60)))
		vars["seconds"] = NewInt(int64(math.Mod(d.Seconds(), 60)))
		vars["millis"] = NewInt(int64(d.Milliseconds() % 1000))
		vars["micros"] = NewInt(int64(d.Microseconds() % 1000))
		vars["nanos"] = NewInt(int64(d.Nanoseconds() % 1000))
	}

	res, err := T(r, locale, Message{
		Key:  key.AsString().Value,
		Vars: vars,
	})
	if err != nil {
		return stringEmpty
	}

	return res.AsString()
}

func (r MarkdownRuntime) PostProcess(value Value, after AfterPostProcessFunc) any {
	return template.HTML(value.AsString().Value)
}
