package router

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/http/middleware"
)

var (
	reParams    = regexp.MustCompile(`:([^/]+)`)
	reLastParam = regexp.MustCompile(`:([^/]+)$`)
)

type ctxKey int

const ctxParams ctxKey = iota

// Route represents a registered route and handler.
type Route struct {
	key      string
	path     string
	pattern  *regexp.Regexp
	parts    []string
	handlers map[string]http.Handler
	methods  []string
}

func (r *Route) String() string {
	return r.path
}

// Replace will create a path based on the route's pattern using the
// replacements given.
//
// Replacements are given as a list of string pairs, where the first in a pair
// is the parameter name starting with a colon, and the second in a pair is the
// string to replace it with.
//
// If a parameter in the route's pattern is missing it will panic.
// Empty replacements will also panic.
func (r *Route) Replace(paramArgPairs ...any) string {
	if len(paramArgPairs)%2 == 1 {
		panic("route path substitution expects an equal number of arguments")
	}

	args := make(map[string]string, len(paramArgPairs)/2)
	for i := 0; i < len(paramArgPairs); i += 2 {
		param := fmt.Sprintf("%v", paramArgPairs[i])
		arg := fmt.Sprintf("%v", paramArgPairs[i+1])

		if !strings.HasPrefix(param, ":") {
			panic(fmt.Sprintf("want argument %v to start with a colon", i))
		}
		if arg == "" {
			panic(fmt.Sprintf("want argument %v to not be empty", i))
		}

		args[param] = arg
	}

	var sb strings.Builder

	seen := make(map[string]struct{})
	for _, part := range r.parts {
		sb.WriteRune('/')

		if strings.HasPrefix(part, ":") {
			arg, ok := args[part]
			if !ok {
				panic(fmt.Sprintf("want an argument for parameter %q in route path to be provided", part))
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

	return sb.String()
}

// ServeMux represents an HTTP router.
type ServeMux struct {
	prefix           string
	middlewares      []middleware.Middleware
	handler          http.Handler
	routes           []*Route
	named            map[string]*Route
	notFound         http.Handler
	methodNotAllowed http.Handler
}

// NewServeMux returns a new serve mux.
func NewServeMux() *ServeMux {
	var sm ServeMux

	sm.handler = http.HandlerFunc(sm.serveHTTP)

	return &sm
}

func (sm *ServeMux) serveHTTP(w http.ResponseWriter, r *http.Request) {
	for _, route := range sm.routes {
		matches := route.pattern.FindStringSubmatch(r.URL.Path)
		if matches == nil {
			continue
		}

		handler := route.handlers[r.Method]
		if handler == nil {
			w.Header().Set("allow", strings.Join(route.methods, ", "))

			switch {
			case r.Method == http.MethodOptions:
				w.WriteHeader(http.StatusNoContent)

			case sm.methodNotAllowed != nil:
				sm.methodNotAllowed.ServeHTTP(w, r)

			default:
				http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			}

			return
		}

		params := make(map[string]string, len(matches))
		names := route.pattern.SubexpNames()
		for i, arg := range matches {
			params[names[i]] = arg
		}

		ctx := context.WithValue(r.Context(), ctxParams, params)

		handler.ServeHTTP(w, r.WithContext(ctx))

		return
	}

	if sm.notFound == nil {
		sm.NotFound(http.HandlerFunc(http.NotFound))
	}

	sm.notFound.ServeHTTP(w, r)
}

// ServeHTTP implements the http.Handler interface.
func (sm *ServeMux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	sm.handler.ServeHTTP(w, r)
}

// Use adds a middleware function to the middleware stack to be called
// before any handlers.
// Middleware registered with this function are called in the order they
// are registered.
func (sm *ServeMux) Use(mw middleware.Middleware) {
	sm.middlewares = append(sm.middlewares, mw)

	sm.handler = middleware.Apply(http.HandlerFunc(sm.serveHTTP), sm.middlewares...)
}

// Prefix will automatically prefix any path patterns that are registered in
// given the route group function with the given prefix.
func (sm *ServeMux) Prefix(prefix string, routeGroup func(*ServeMux)) {
	originalPrefix := sm.prefix
	sm.prefix += prefix

	routeGroup(sm)

	sm.prefix = originalPrefix
}

func (sm *ServeMux) Route(name string) *Route {
	return sm.named[name]
}

func (sm *ServeMux) Path(key string, paramArgPairs ...any) string {
	route := sm.Route(key)
	if route == nil {
		panic(fmt.Sprintf("route %q does not exist", key))
	}

	if len(paramArgPairs) != 0 {
		return route.Replace(paramArgPairs...)
	}

	str := route.String()
	if strings.Contains(str, "/:") {
		panic(fmt.Sprintf("route %q must use the replace method to replace parameters", key))
	}

	return str
}

func (sm *ServeMux) route(method string, path string, handler http.Handler, names ...string) *Route {
	method = strings.ToUpper(method)
	path = sm.prefix + path
	parts := strings.Split(strings.TrimPrefix(path, "/"), "/")
	pattern := regexp.QuoteMeta(path)

	if pattern == "" {
		panic("route must not be empty")
	}

	pattern = reLastParam.ReplaceAllString(pattern, `(?P<$1>.*?)`)
	pattern = reParams.ReplaceAllString(pattern, `(?P<$1>[^/]+?)`)
	pattern = "^" + pattern + "$"

	key := reParams.ReplaceAllString(path, "*")

	for _, route := range sm.routes {
		if route.key != key {
			continue
		}

		if _, ok := route.handlers[method]; ok {
			panic(fmt.Sprintf("duplicate route registration for %q and %q (%v)", path, route.path, method))
		}

		route.handlers[method] = handler
		route.methods = append(route.methods, method)

		if len(names) != 0 {
			if sm.named == nil {
				sm.named = make(map[string]*Route)
			}

			for _, name := range names {
				if _, ok := sm.named[name]; ok {
					panic(fmt.Sprintf("duplicate route name %q", name))
				}

				sm.named[name] = route
			}
		}

		return route
	}

	compiled := regexp.MustCompile(pattern)

	seen := make(map[string]struct{})
	for _, name := range compiled.SubexpNames() {
		if _, ok := seen[name]; ok {
			panic(fmt.Sprintf("duplicate parameter name %q in route %q", name, path))
		}

		seen[name] = struct{}{}
	}

	route := &Route{
		key:      key,
		path:     path,
		pattern:  compiled,
		parts:    parts,
		handlers: map[string]http.Handler{method: handler},
		methods:  []string{http.MethodOptions},
	}

	sm.routes = append(sm.routes, route)

	if len(names) != 0 {
		if sm.named == nil {
			sm.named = make(map[string]*Route)
		}

		for _, name := range names {
			sm.named[name] = route
		}
	}

	return route
}

// OptionsHandler registers a handler that can be used to serve any OPTIONS
// request matching the given path pattern.
func (sm *ServeMux) OptionsHandler(path string, handler http.Handler, names ...string) *Route {
	return sm.route(http.MethodOptions, path, handler, names...)
}

// Options registers a handler that can be used to serve any OPTIONS
// request matching the given path pattern.
func (sm *ServeMux) Options(path string, handler http.HandlerFunc, names ...string) *Route {
	return sm.OptionsHandler(path, http.HandlerFunc(handler), names...)
}

// ConnectHandler registers a handler that can be used to serve any CONNECT
// request matching the given path pattern.
func (sm *ServeMux) ConnectHandler(path string, handler http.Handler, names ...string) *Route {
	return sm.route(http.MethodConnect, path, handler, names...)
}

// Connect registers a handler that can be used to serve any CONNECT
// request matching the given path pattern.
func (sm *ServeMux) Connect(path string, handler http.HandlerFunc, names ...string) *Route {
	return sm.ConnectHandler(path, http.HandlerFunc(handler), names...)
}

// TraceHandler registers a handler that can be used to serve any TRACE
// request matching the given path pattern.
func (sm *ServeMux) TraceHandler(path string, handler http.Handler, names ...string) *Route {
	return sm.route(http.MethodTrace, path, handler, names...)
}

// Trace registers a handler that can be used to serve any TRACE
// request matching the given path pattern.
func (sm *ServeMux) Trace(path string, handler http.HandlerFunc, names ...string) *Route {
	return sm.TraceHandler(path, http.HandlerFunc(handler), names...)
}

// HeadHandler registers a handler that can be used to serve any HEAD
// request matching the given path pattern.
func (sm *ServeMux) HeadHandler(path string, handler http.Handler, names ...string) *Route {
	return sm.route(http.MethodHead, path, handler, names...)
}

// Head registers a handler that can be used to serve any HEAD
// request matching the given path pattern.
func (sm *ServeMux) Head(path string, handler http.HandlerFunc, names ...string) *Route {
	return sm.HeadHandler(path, http.HandlerFunc(handler), names...)
}

// GetHandler registers a handler that can be used to serve any GET
// request matching the given path pattern.
func (sm *ServeMux) GetHandler(path string, handler http.Handler, names ...string) *Route {
	return sm.route(http.MethodGet, path, handler, names...)
}

// Get registers a handler that can be used to serve any GET
// request matching the given path pattern.
func (sm *ServeMux) Get(path string, handler http.HandlerFunc, names ...string) *Route {
	return sm.GetHandler(path, http.HandlerFunc(handler), names...)
}

// PostHandler registers a handler that can be used to serve any POST
// request matching the given path pattern.
func (sm *ServeMux) PostHandler(path string, handler http.Handler, names ...string) *Route {
	return sm.route(http.MethodPost, path, handler, names...)
}

// Post registers a handler that can be used to serve any POST
// request matching the given path pattern.
func (sm *ServeMux) Post(path string, handler http.HandlerFunc, names ...string) *Route {
	return sm.PostHandler(path, http.HandlerFunc(handler), names...)
}

// PutHandler registers a handler that can be used to serve any PUT
// request matching the given path pattern.
func (sm *ServeMux) PutHandler(path string, handler http.Handler, names ...string) *Route {
	return sm.route(http.MethodPut, path, handler, names...)
}

// Put registers a handler that can be used to serve any PUT
// request matching the given path pattern.
func (sm *ServeMux) Put(path string, handler http.HandlerFunc, names ...string) *Route {
	return sm.PutHandler(path, http.HandlerFunc(handler), names...)
}

// PatchHandler registers a handler that can be used to serve any PATCH
// request matching the given path pattern.
func (sm *ServeMux) PatchHandler(path string, handler http.Handler, names ...string) *Route {
	return sm.route(http.MethodPatch, path, handler, names...)
}

// Patch registers a handler that can be used to serve any PATCH
// request matching the given path pattern.
func (sm *ServeMux) Patch(path string, handler http.HandlerFunc, names ...string) *Route {
	return sm.PatchHandler(path, http.HandlerFunc(handler), names...)
}

// DeleteHandler registers a handler that can be used to serve any DELETE
// request matching the given path pattern.
func (sm *ServeMux) DeleteHandler(path string, handler http.Handler, names ...string) *Route {
	return sm.route(http.MethodDelete, path, handler, names...)
}

// Delete registers a handler that can be used to serve any DELETE
// request matching the given path pattern.
func (sm *ServeMux) Delete(path string, handler http.HandlerFunc, names ...string) *Route {
	return sm.DeleteHandler(path, http.HandlerFunc(handler), names...)
}

// AnyHandler registers a handler that can be used to serve any request matching
// the given path pattern.
func (sm *ServeMux) AnyHandler(path string, handler http.Handler, names ...string) *Route {
	route := sm.OptionsHandler(path, handler, names...)
	sm.ConnectHandler(path, handler)
	sm.TraceHandler(path, handler)
	sm.HeadHandler(path, handler)
	sm.GetHandler(path, handler)
	sm.PostHandler(path, handler)
	sm.PutHandler(path, handler)
	sm.PatchHandler(path, handler)
	sm.DeleteHandler(path, handler)

	return route
}

// Any registers a handler that can be used to serve any request matching
// the given path pattern.
func (sm *ServeMux) Any(path string, handler http.HandlerFunc, names ...string) *Route {
	return sm.AnyHandler(path, http.HandlerFunc(handler), names...)
}

// NotFoundHandler registers a handler to be used when an HTTP not found error
// is triggered.
func (sm *ServeMux) NotFoundHandler(handler http.Handler) {
	sm.notFound = handler
}

// NotFound registers a handler to be used when an HTTP not found error
// is triggered.
func (sm *ServeMux) NotFound(handler http.HandlerFunc) {
	sm.NotFoundHandler(http.HandlerFunc(handler))
}

// MethodNotAllowedHandler registers a handler to be used when an HTTP method
// not allowed error is triggered.
func (sm *ServeMux) MethodNotAllowedHandler(handler http.Handler) {
	sm.methodNotAllowed = handler
}

// MethodNotAllowed registers a handler to be used when an HTTP method
// not allowed error is triggered.
func (sm *ServeMux) MethodNotAllowed(handler http.HandlerFunc) {
	sm.MethodNotAllowedHandler(http.HandlerFunc(handler))
}

// Rewrite will register a new route that rewrites the source path pattern to
// the destination path pattern, and then attempts to find a handler that
// matches the new pattern instead.
//
// Any parameters used in the source path will have their value replaced into
// the destination path where the same parameter is used.
// For example, the patterns: "/:foo/greet"; "/:foo/world", will rewrite the
// source "/hello/greet" to "/hello/world".
func (sm *ServeMux) Rewrite(method, src, dst string) {
	originalPrefix := sm.prefix
	sm.prefix = ""

	sm.route(method, src, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		dst := dst

		params, ok := r.Context().Value(ctxParams).(map[string]string)
		if ok {
			keys := make([]string, 0, len(params))
			for key := range params {
				keys = append(keys, key)
			}

			sort.Slice(keys, func(i, j int) bool {
				// Reverse string length sort so the longest key comes first
				return utf8.RuneCountInString(keys[j]) < utf8.RuneCountInString(keys[i])
			})

			for _, key := range keys {
				dst = strings.ReplaceAll(dst, ":"+key, params[key])
			}
		}

		r.URL.Path = dst

		sm.ServeHTTP(w, r)
	}))

	sm.prefix = originalPrefix
}

// Redirect will create a new handler for the given source path that will
// redirect the request to the given destination path using the code.
//
// Any parameters used in the source path will have their value replaced into
// the destination path where the same parameter is used.
// For example, the patterns: "/:foo/greet"; "/:foo/world", will redirect the
// source "/hello/greet" to "/hello/world".
func (sm *ServeMux) Redirect(method, src, dst string, code int) {
	originalPrefix := sm.prefix
	sm.prefix = ""

	sm.route(method, src, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		dst := dst

		params, ok := r.Context().Value(ctxParams).(map[string]string)
		if ok {
			keys := make([]string, 0, len(params))
			for key := range params {
				keys = append(keys, key)
			}

			sort.Slice(keys, func(i, j int) bool {
				// Reverse string length sort so the longest key comes first
				return utf8.RuneCountInString(keys[j]) < utf8.RuneCountInString(keys[i])
			})

			for _, key := range keys {
				dst = strings.ReplaceAll(dst, ":"+key, params[key])
			}
		}

		http.Redirect(w, r, dst, code)
	}))

	sm.prefix = originalPrefix
}

// URLParam returns the string value associated with the given parameter name in
// the given request URL.
// If the parameter name is not found then it panics.
func URLParam(r *http.Request, name string) string {
	value, ok := r.Context().Value(ctxParams).(map[string]string)[name]
	if !ok {
		panic(fmt.Sprintf("required url parameter %q is missing for %q", name, r.URL))
	}

	return value
}

// URLParamAs returns the value associated with the given parameter name in
// the given request URL after attempting to convert it to the given type T.
// If the parameter name is not found then it panics.
func URLParamAs[T any](r *http.Request, name string) (T, error) {
	str := URLParam(r, name)

	var res T
	as := reflect.ValueOf(&res).Elem()

	var err error
	switch typ := as.Type(); typ.Kind() {
	case reflect.Bool:
		as.SetBool(str == "1" || str == "checked")

	case reflect.Float32:
		var value float64
		if str != "" {
			value, err = strconv.ParseFloat(str, 32)
			if err != nil {
				return res, errors.Tracef(err)
			}
		}

		as.SetFloat(value)

	case reflect.Float64:
		var value float64
		if str != "" {
			value, err = strconv.ParseFloat(str, 64)
			if err != nil {
				return res, errors.Tracef(err)
			}
		}

		as.SetFloat(value)

	case reflect.Int8:
		var value int64
		if str != "" {
			value, err = strconv.ParseInt(str, 10, 8)
			if err != nil {
				return res, errors.Tracef(err)
			}
		}

		as.SetInt(value)

	case reflect.Int16:
		var value int64
		if str != "" {
			value, err = strconv.ParseInt(str, 10, 16)
			if err != nil {
				return res, errors.Tracef(err)
			}
		}

		as.SetInt(value)

	case reflect.Int32:
		var value int64
		if str != "" {
			value, err = strconv.ParseInt(str, 10, 32)
			if err != nil {
				return res, errors.Tracef(err)
			}
		}

		as.SetInt(value)

	case reflect.Int64:
		var value int64
		if str != "" {
			value, err = strconv.ParseInt(str, 10, 64)
			if err != nil {
				return res, errors.Tracef(err)
			}
		}

		as.SetInt(value)

	case reflect.Int:
		var value int64
		if str != "" {
			value, err = strconv.ParseInt(str, 10, 64)
			if err != nil {
				return res, errors.Tracef(err)
			}
		}

		as.SetInt(value)

	case reflect.Uint8:
		var value uint64
		if str != "" {
			value, err = strconv.ParseUint(str, 10, 8)
			if err != nil {
				return res, errors.Tracef(err)
			}
		}

		as.SetUint(value)

	case reflect.Uint16:
		var value uint64
		if str != "" {
			value, err = strconv.ParseUint(str, 10, 16)
			if err != nil {
				return res, errors.Tracef(err)
			}
		}

		as.SetUint(value)

	case reflect.Uint32:
		var value uint64
		if str != "" {
			value, err = strconv.ParseUint(str, 10, 32)
			if err != nil {
				return res, errors.Tracef(err)
			}
		}

		as.SetUint(value)

	case reflect.Uint64:
		var value uint64
		if str != "" {
			value, err = strconv.ParseUint(str, 10, 64)
			if err != nil {
				return res, errors.Tracef(err)
			}
		}

		as.SetUint(value)

	case reflect.Uint:
		var value uint64
		if str != "" {
			value, err = strconv.ParseUint(str, 10, 64)
			if err != nil {
				return res, errors.Tracef(err)
			}
		}

		as.SetUint(value)

	case reflect.String:
		as.SetString(str)

	default:
		if typ == reflect.TypeOf([]byte(nil)) {
			as.SetBytes([]byte(str))
		} else {
			return res, errors.Tracef("unsupported type %v", typ)
		}
	}

	return res, nil
}
