package i18n

import (
	"fmt"
	"html/template"
	"strings"
)

type JSRuntime struct {
	MarkdownRuntime
}

func (r JSRuntime) Kind() string {
	return "js"
}

func (r JSRuntime) Bold(value Value) RawString {
	s := value.AsString().Value
	if s == "" {
		return rawStringEmpty
	}

	return NewRawString("<b>" + template.JSEscapeString(s) + "</b>")
}

func (r JSRuntime) Italic(value Value) RawString {
	s := value.AsString().Value
	if s == "" {
		return rawStringEmpty
	}

	return NewRawString("<i>" + template.JSEscapeString(s) + "</i>")
}

func (r JSRuntime) Link(label, href, target Value) RawString {
	_label := template.JSEscapeString(label.AsString().Value)
	_href := template.JSEscapeString(href.AsString().Value)

	var link string
	if _target := target.AsString().Value; _target != "" {
		link = fmt.Sprintf("<a href=%q target=%q>%v</a>", _href, _target, _label)
	} else {
		link = fmt.Sprintf("<a href=%q>%v</a>", _href, _label)
	}

	return NewRawString(link)
}

func (r JSRuntime) PostProcess(value Value, after AfterPostProcessFunc) any {
	var sb strings.Builder
	for _, v := range value.AsSlice() {
		var str string
		if v.Type() == TypeRawString {
			str = v.AsString().Value
		} else {
			str = template.JSEscapeString(v.AsString().Value)
		}

		sb.WriteString(str)
	}

	return template.JS(sb.String())
}
