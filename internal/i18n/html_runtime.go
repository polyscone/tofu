package i18n

import (
	"fmt"
	"html/template"
	"strings"
)

type HTMLRuntime struct {
	MarkdownRuntime
}

func (r HTMLRuntime) Kind() string {
	return "html"
}

func (r HTMLRuntime) Bold(value Value) RawString {
	s := value.AsString().Value
	if s == "" {
		return rawStringEmpty
	}

	return NewRawString("<b>" + template.HTMLEscapeString(s) + "</b>")
}

func (r HTMLRuntime) Italic(value Value) RawString {
	s := value.AsString().Value
	if s == "" {
		return rawStringEmpty
	}

	return NewRawString("<i>" + template.HTMLEscapeString(s) + "</i>")
}

func (r HTMLRuntime) Link(label, href, target Value) RawString {
	_label := template.HTMLEscapeString(label.AsString().Value)
	_href := template.HTMLEscapeString(href.AsString().Value)

	var link string
	if _target := target.AsString().Value; _target != "" {
		link = fmt.Sprintf("<a href=%q target=%q>%v</a>", _href, _target, _label)
	} else {
		link = fmt.Sprintf("<a href=%q>%v</a>", _href, _label)
	}

	return NewRawString(link)
}

func (r HTMLRuntime) PostProcess(value Value, after AfterPostProcessFunc) any {
	var sb strings.Builder
	for _, v := range value.AsSlice() {
		var str string
		if v.Type() == TypeRawString {
			str = v.AsString().Value
		} else {
			str = template.HTMLEscapeString(v.AsString().Value)
		}

		sb.WriteString(str)
	}

	str := sb.String()
	if after != nil {
		str = after(str)
	}

	return template.HTML(str)
}
