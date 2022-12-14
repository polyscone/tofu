package errors

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Sprint returns the given error's string value by calling Error(), unless it
// is a Trace, in which case it will return String().
func Sprint(err error) string {
	if trace, ok := err.(Trace); ok {
		return trace.String()
	}

	return err.Error()
}

// SprintJSON returns the given error as a JSON string.
func SprintJSON(err error) string {
	b, err := json.Marshal(err)
	if err != nil {
		b = []byte(err.Error())
	}

	return string(b)
}

func fprintln(sb *strings.Builder, value any, prefix string) {
	switch value := value.(type) {
	case []Trace:
		for i, trace := range value {
			symbol := " ├ "
			if i == len(value)-1 {
				symbol = " └ "
			}

			fprintln(sb, trace.Frame, prefix+symbol)

			if i == len(value)-1 {
				symbol = "   "
			} else {
				symbol = " │ "
			}

			for key, err := range trace.fields {
				sb.WriteString(fmt.Sprintf("%v      %v%v: %v\n", symbol, prefix, key, err))

				fprintln(sb, StackTrace(err), symbol+"      "+prefix)
			}
		}

	case Trace:
		sb.WriteString(fmt.Sprintf("%v%v\n", prefix, value))

		fprintln(sb, StackTrace(value), prefix)

	case Frame:
		funcParts := strings.Split(value.FuncName, "/")
		funcName := funcParts[len(funcParts)-1]

		sb.WriteString(fmt.Sprintf("%v%v:%v (%v)\n", prefix, value.File, value.Line, funcName))

	default:
		if value != nil {
			sb.WriteString(fmt.Sprintf("%v%v\n", prefix, value))
		}
	}
}

func sprintJSON(value any) string {
	switch value := value.(type) {
	case []Trace:
		var traces []string
		for _, trace := range value {
			var fields []string
			for key, err := range trace.fields {
				stack := sprintJSON(StackTrace(err))
				if stack != "" {
					stack = "," + stack
				}

				fields = append(fields, fmt.Sprintf(`%q:{"error":%q%v}`, key, err, stack))
			}

			var fieldsJSON string
			if fields != nil {
				fieldsJSON = fmt.Sprintf(`,"fields":{%v}`, strings.Join(fields, ","))
			}

			frame := sprintJSON(trace.Frame)

			traces = append(traces, fmt.Sprintf(`{%v%v}`, frame, fieldsJSON))
		}

		if traces == nil {
			return ""
		}

		return fmt.Sprintf(`"stack":[%v]`, strings.Join(traces, ","))

	case Trace:
		stack := sprintJSON(StackTrace(value))
		if stack != "" {
			stack = "," + stack
		}

		return fmt.Sprintf(`{"error":%q%v}`, value, stack)

	case Frame:
		return fmt.Sprintf(`"file":%q,"line":%v,"function":%q`, value.File, value.Line, value.FuncName)

	default:
		if value != nil {
			b, err := json.Marshal(value)
			if err != nil {
				b = []byte(err.Error())
			}

			return fmt.Sprintf(`{"error":%q}`, b)
		}

		return ""
	}
}
