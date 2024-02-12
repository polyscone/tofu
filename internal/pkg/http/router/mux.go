package router

import (
	"fmt"
	"net/http"
	"reflect"
	"slices"
	"strconv"
	"strings"

	"github.com/polyscone/tofu/internal/pkg/http/middleware"
)

const (
	paramStart = "{"
	paramEnd   = "}"
)

type ServeMux struct {
	*http.ServeMux
	middlewares []middleware.Middleware
	befores     []middleware.Middleware
	handler     http.Handler
	groups      int
	named       map[string]string
}

func NewServeMux() *ServeMux {
	mux := ServeMux{
		ServeMux: http.NewServeMux(),
		named:    make(map[string]string),
	}

	mux.handler = http.HandlerFunc(mux.ServeMux.ServeHTTP)

	return &mux
}

func (mux *ServeMux) Use(mw middleware.Middleware) {
	// We panic on calls to Use if we're still in a call to Group because
	// we don't want the user to think that a call to Use will affect only the
	// current group
	if mux.groups != 0 {
		panic("cannot call ServeMux.Use within a call to ServeMux.Group")
	}

	mux.middlewares = append(mux.middlewares, mw)

	mux.handler = middleware.Apply(http.HandlerFunc(mux.ServeMux.ServeHTTP), mux.middlewares...)
}

func (mux *ServeMux) Before(mw middleware.Middleware) {
	mux.befores = append(mux.befores, mw)
}

func (mux *ServeMux) Group(routeGroup func(mux *ServeMux)) {
	mux.groups++
	originalBefores := slices.Clone(mux.befores)

	routeGroup(mux)

	mux.befores = originalBefores
	mux.groups--
}

func (mux *ServeMux) Named(name, pattern string) {
	if _, ok := mux.named[name]; ok {
		panic(fmt.Sprintf("duplicate name %q", name))
	}

	mux.named[name] = pattern
}

func (mux *ServeMux) Handle(pattern string, handler http.Handler, names ...string) {
	if len(names) > 0 {
		for _, name := range names {
			mux.Named(name, pattern)
		}
	}

	if len(mux.befores) > 0 {
		handler = middleware.Apply(handler, mux.befores...)
	}

	mux.ServeMux.Handle(pattern, handler)
}

func (mux *ServeMux) HandleFunc(pattern string, handler func(http.ResponseWriter, *http.Request), names ...string) {
	mux.Handle(pattern, http.HandlerFunc(handler), names...)
}

func (mux *ServeMux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	mux.handler.ServeHTTP(w, r)
}

func (mux *ServeMux) Path(name string, paramArgPairs ...any) string {
	pattern := mux.named[name]
	if pattern == "" {
		panic(fmt.Sprintf("named path %q does not exist", name))
	}

	pattern = strings.ReplaceAll(pattern, "{$}", "")
	pattern = strings.TrimPrefix(pattern, http.MethodOptions+" ")
	pattern = strings.TrimPrefix(pattern, http.MethodConnect+" ")
	pattern = strings.TrimPrefix(pattern, http.MethodTrace+" ")
	pattern = strings.TrimPrefix(pattern, http.MethodHead+" ")
	pattern = strings.TrimPrefix(pattern, http.MethodGet+" ")
	pattern = strings.TrimPrefix(pattern, http.MethodPost+" ")
	pattern = strings.TrimPrefix(pattern, http.MethodPut+" ")
	pattern = strings.TrimPrefix(pattern, http.MethodPatch+" ")
	pattern = strings.TrimPrefix(pattern, http.MethodDelete+" ")

	if len(paramArgPairs) > 0 {
		if len(paramArgPairs)%2 == 1 {
			panic("named path pattern substitution expects an equal number of arguments")
		}

		args := make(map[string]string, len(paramArgPairs)/2)
		for i := 0; i < len(paramArgPairs); i += 2 {
			param := fmt.Sprintf("%v", paramArgPairs[i])
			arg := fmt.Sprintf("%v", paramArgPairs[i+1])

			if !strings.HasPrefix(param, paramStart) || !strings.HasSuffix(param, paramEnd) {
				panic(fmt.Sprintf("want argument %v to be wrapped in curly braces", i))
			}
			if arg == "" {
				panic(fmt.Sprintf("want argument %v to not be empty", i))
			}

			args[param] = arg
		}

		var sb strings.Builder

		parts := strings.Split(strings.TrimPrefix(pattern, "/"), "/")
		seen := make(map[string]struct{})
		for _, part := range parts {
			sb.WriteRune('/')

			if strings.HasPrefix(part, paramStart) {
				arg, ok := args[part]
				if !ok {
					panic(fmt.Sprintf("want an argument for parameter %q in named path pattern to be provided", part))
				}

				seen[part] = struct{}{}
				part = arg
			}

			sb.WriteString(part)
		}

		if len(seen) != len(args) {
			var unknowns []string
			for param := range args {
				if _, ok := seen[param]; ok {
					continue
				}

				unknowns = append(unknowns, param)
			}

			panic(fmt.Sprintf("unknown parameters: %q", unknowns))
		}

		pattern = sb.String()
	}

	if strings.Contains(pattern, "/"+paramStart) {
		panic(fmt.Sprintf("named path %q requires parameters to be replaced", name))
	}

	return pattern
}

func PathValueAs[T any](r *http.Request, name string) (T, bool) {
	var res T
	var err error

	str := r.PathValue(name)
	as := reflect.ValueOf(&res).Elem()
	switch typ := as.Type(); typ.Kind() {
	case reflect.Bool:
		as.SetBool(str == "1" || str == "on")

	case reflect.Float32:
		var value float64
		if str != "" {
			value, err = strconv.ParseFloat(str, 32)
			if err != nil {
				return res, false
			}
		}

		as.SetFloat(value)

	case reflect.Float64:
		var value float64
		if str != "" {
			value, err = strconv.ParseFloat(str, 64)
			if err != nil {
				return res, false
			}
		}

		as.SetFloat(value)

	case reflect.Int8:
		var value int64
		if str != "" {
			value, err = strconv.ParseInt(str, 10, 8)
			if err != nil {
				return res, false
			}
		}

		as.SetInt(value)

	case reflect.Int16:
		var value int64
		if str != "" {
			value, err = strconv.ParseInt(str, 10, 16)
			if err != nil {
				return res, false
			}
		}

		as.SetInt(value)

	case reflect.Int32:
		var value int64
		if str != "" {
			value, err = strconv.ParseInt(str, 10, 32)
			if err != nil {
				return res, false
			}
		}

		as.SetInt(value)

	case reflect.Int64:
		var value int64
		if str != "" {
			value, err = strconv.ParseInt(str, 10, 64)
			if err != nil {
				return res, false
			}
		}

		as.SetInt(value)

	case reflect.Int:
		var value int64
		if str != "" {
			value, err = strconv.ParseInt(str, 10, 64)
			if err != nil {
				return res, false
			}
		}

		as.SetInt(value)

	case reflect.Uint8:
		var value uint64
		if str != "" {
			value, err = strconv.ParseUint(str, 10, 8)
			if err != nil {
				return res, false
			}
		}

		as.SetUint(value)

	case reflect.Uint16:
		var value uint64
		if str != "" {
			value, err = strconv.ParseUint(str, 10, 16)
			if err != nil {
				return res, false
			}
		}

		as.SetUint(value)

	case reflect.Uint32:
		var value uint64
		if str != "" {
			value, err = strconv.ParseUint(str, 10, 32)
			if err != nil {
				return res, false
			}
		}

		as.SetUint(value)

	case reflect.Uint64:
		var value uint64
		if str != "" {
			value, err = strconv.ParseUint(str, 10, 64)
			if err != nil {
				return res, false
			}
		}

		as.SetUint(value)

	case reflect.Uint:
		var value uint64
		if str != "" {
			value, err = strconv.ParseUint(str, 10, 64)
			if err != nil {
				return res, false
			}
		}

		as.SetUint(value)

	case reflect.String:
		as.SetString(str)

	default:
		switch typ {
		case reflect.TypeOf([]byte(nil)):
			as.SetBytes([]byte(str))

		default:
			panic(fmt.Sprintf("unsupported conversion type %v", typ))
		}
	}

	return res, true
}
