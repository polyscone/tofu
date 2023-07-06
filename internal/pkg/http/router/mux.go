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

	"github.com/polyscone/tofu/internal/pkg/http/middleware"
)

var (
	reParams     = regexp.MustCompile(`:([^/]+)`)
	reLastParam  = regexp.MustCompile(`:([^/]+)$`)
	reMultiSlash = regexp.MustCompile(`//+`)
)

type ctxKey int

const ctxParams ctxKey = iota

type BeforeHookFunc func(w http.ResponseWriter, r *http.Request) bool

type BeforeHook struct {
	pattern *regexp.Regexp
	fn      BeforeHookFunc
}

// Route represents a registered route and handler.
type Route struct {
	key      string
	host     string
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
	host             string
	prefix           string
	middlewares      []middleware.Middleware
	handler          http.Handler
	befores          []*BeforeHook
	routes           []*Route
	named            map[string]*Route
	notFound         http.Handler
	methodNotAllowed http.Handler
}

// NewServeMux returns a new serve mux.
func NewServeMux() *ServeMux {
	var mux ServeMux

	mux.handler = http.HandlerFunc(mux.serveHTTP)

	return &mux
}

func (mux *ServeMux) serveHTTP(w http.ResponseWriter, r *http.Request) {
	for _, route := range mux.routes {
		if route.host != "" && route.host != r.Host {
			continue
		}

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

			case mux.methodNotAllowed != nil:
				mux.methodNotAllowed.ServeHTTP(w, r)

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
		r := r.WithContext(ctx)

		for _, hook := range mux.befores {
			if !hook.pattern.MatchString(r.URL.Path) {
				continue
			}

			if ok := hook.fn(w, r); !ok {
				return
			}
		}

		handler.ServeHTTP(w, r)

		return
	}

	if mux.notFound == nil {
		mux.NotFound(http.HandlerFunc(http.NotFound))
	}

	mux.notFound.ServeHTTP(w, r)
}

// ServeHTTP implements the http.Handler interface.
func (mux *ServeMux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	mux.handler.ServeHTTP(w, r)
}

// Use adds a middleware function to the middleware stack to be called
// before any handlers.
// Middleware registered with this function are called in the order they
// are registered.
func (mux *ServeMux) Use(mw middleware.Middleware) {
	mux.middlewares = append(mux.middlewares, mw)

	mux.handler = middleware.Apply(http.HandlerFunc(mux.serveHTTP), mux.middlewares...)
}

func (mux *ServeMux) Before(before BeforeHookFunc, paths ...string) {
	if len(paths) == 0 {
		if mux.prefix == "" {
			panic("before hooks without exact paths must be registered in a non-empty prefix")
		}

		pattern := regexp.QuoteMeta(mux.prefix)
		pattern = reParams.ReplaceAllString(pattern, `([^/]+)`)
		pattern = "^" + pattern

		compiled := regexp.MustCompile(pattern)

		mux.befores = append(mux.befores, &BeforeHook{
			pattern: compiled,
			fn:      before,
		})
	} else {
		for _, path := range paths {
			pattern := regexp.QuoteMeta(path)
			pattern = reParams.ReplaceAllString(pattern, `([^/]+)`)
			pattern = "^" + pattern + "$"

			compiled := regexp.MustCompile(pattern)

			mux.befores = append(mux.befores, &BeforeHook{
				pattern: compiled,
				fn:      before,
			})
		}
	}
}

// Host will scope handlers to a specific request host value.
func (mux *ServeMux) Host(host string, routeGroup func(mux *ServeMux)) {
	if mux.host != "" {
		panic("nested calls to ServeMux.Host()")
	}

	mux.host = host

	routeGroup(mux)

	mux.host = ""
}

// Prefix will automatically prefix any path patterns that are registered in
// given the route group function with the given prefix.
func (mux *ServeMux) Prefix(prefix string, routeGroup func(mux *ServeMux)) {
	originalPrefix := mux.prefix
	mux.prefix += prefix

	routeGroup(mux)

	mux.prefix = originalPrefix
}

func (mux *ServeMux) CurrentPrefix() string {
	return mux.prefix + "/"
}

func (mux *ServeMux) CurrentPath() string {
	return mux.prefix
}

func (mux *ServeMux) Name(name string) {
	route := &Route{path: mux.prefix}

	mux.nameRoute(route, name)
}

func (mux *ServeMux) Route(name string) *Route {
	return mux.named[name]
}

func (mux *ServeMux) Path(name string, paramArgPairs ...any) string {
	route := mux.Route(name)
	if route == nil {
		panic(fmt.Sprintf("route %q does not exist", name))
	}

	if len(paramArgPairs) > 0 {
		return route.Replace(paramArgPairs...)
	}

	str := route.String()
	if strings.Contains(str, "/:") {
		panic(fmt.Sprintf("route %q must use the replace method to replace parameters", name))
	}

	return str
}

func (mux *ServeMux) nameRoute(route *Route, names ...string) {
	if len(names) > 0 {
		if mux.named == nil {
			mux.named = make(map[string]*Route)
		}

		for _, name := range names {
			if _, ok := mux.named[name]; ok {
				panic(fmt.Sprintf("duplicate route name %q", name))
			}

			mux.named[name] = route
		}
	}
}

func (mux *ServeMux) route(method string, path string, handler http.Handler, names ...string) *Route {
	method = strings.ToUpper(method)

	if path == "" {
		panic("route path must not be empty")
	}

	if mux.prefix != "" && path == "/" {
		path = ""
	}

	path = mux.prefix + path
	path = reMultiSlash.ReplaceAllString(path, "/")
	parts := strings.Split(strings.TrimPrefix(path, "/"), "/")

	if path == "" {
		panic("route must not be empty")
	}

	var restGroup string
	if rest := reLastParam.FindString(path); strings.HasSuffix(rest, "*") {
		path = strings.TrimSuffix(path, "*")
		restGroup = `(?P<$1>.*)`
	} else {
		restGroup = `(?P<$1>[^/]*)`
	}

	key := mux.host + "/" + reParams.ReplaceAllString(path, "*")

	for _, route := range mux.routes {
		if route.key != key {
			continue
		}

		if _, ok := route.handlers[method]; ok {
			panic(fmt.Sprintf("duplicate route registration for %q and %q (%v)", path, route.path, method))
		}

		route.handlers[method] = handler
		route.methods = append(route.methods, method)

		mux.nameRoute(route, names...)

		return route
	}

	pattern := regexp.QuoteMeta(path)
	pattern = reLastParam.ReplaceAllString(pattern, restGroup)
	pattern = reParams.ReplaceAllString(pattern, `(?P<$1>[^/]+)`)
	pattern = "^" + pattern + "$"

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
		host:     mux.host,
		path:     path,
		pattern:  compiled,
		parts:    parts,
		handlers: map[string]http.Handler{method: handler},
		methods:  []string{http.MethodOptions},
	}

	mux.routes = append(mux.routes, route)

	mux.nameRoute(route, names...)

	return route
}

// OptionsHandler registers a handler that can be used to serve any OPTIONS
// request matching the given path pattern.
func (mux *ServeMux) OptionsHandler(path string, handler http.Handler, names ...string) *Route {
	return mux.route(http.MethodOptions, path, handler, names...)
}

// Options registers a handler that can be used to serve any OPTIONS
// request matching the given path pattern.
func (mux *ServeMux) Options(path string, handler http.HandlerFunc, names ...string) *Route {
	return mux.OptionsHandler(path, http.HandlerFunc(handler), names...)
}

// ConnectHandler registers a handler that can be used to serve any CONNECT
// request matching the given path pattern.
func (mux *ServeMux) ConnectHandler(path string, handler http.Handler, names ...string) *Route {
	return mux.route(http.MethodConnect, path, handler, names...)
}

// Connect registers a handler that can be used to serve any CONNECT
// request matching the given path pattern.
func (mux *ServeMux) Connect(path string, handler http.HandlerFunc, names ...string) *Route {
	return mux.ConnectHandler(path, http.HandlerFunc(handler), names...)
}

// TraceHandler registers a handler that can be used to serve any TRACE
// request matching the given path pattern.
func (mux *ServeMux) TraceHandler(path string, handler http.Handler, names ...string) *Route {
	return mux.route(http.MethodTrace, path, handler, names...)
}

// Trace registers a handler that can be used to serve any TRACE
// request matching the given path pattern.
func (mux *ServeMux) Trace(path string, handler http.HandlerFunc, names ...string) *Route {
	return mux.TraceHandler(path, http.HandlerFunc(handler), names...)
}

// HeadHandler registers a handler that can be used to serve any HEAD
// request matching the given path pattern.
func (mux *ServeMux) HeadHandler(path string, handler http.Handler, names ...string) *Route {
	return mux.route(http.MethodHead, path, handler, names...)
}

// Head registers a handler that can be used to serve any HEAD
// request matching the given path pattern.
func (mux *ServeMux) Head(path string, handler http.HandlerFunc, names ...string) *Route {
	return mux.HeadHandler(path, http.HandlerFunc(handler), names...)
}

// GetHandler registers a handler that can be used to serve any GET
// request matching the given path pattern.
func (mux *ServeMux) GetHandler(path string, handler http.Handler, names ...string) *Route {
	return mux.route(http.MethodGet, path, handler, names...)
}

// Get registers a handler that can be used to serve any GET
// request matching the given path pattern.
func (mux *ServeMux) Get(path string, handler http.HandlerFunc, names ...string) *Route {
	return mux.GetHandler(path, http.HandlerFunc(handler), names...)
}

// PostHandler registers a handler that can be used to serve any POST
// request matching the given path pattern.
func (mux *ServeMux) PostHandler(path string, handler http.Handler, names ...string) *Route {
	return mux.route(http.MethodPost, path, handler, names...)
}

// Post registers a handler that can be used to serve any POST
// request matching the given path pattern.
func (mux *ServeMux) Post(path string, handler http.HandlerFunc, names ...string) *Route {
	return mux.PostHandler(path, http.HandlerFunc(handler), names...)
}

// PutHandler registers a handler that can be used to serve any PUT
// request matching the given path pattern.
func (mux *ServeMux) PutHandler(path string, handler http.Handler, names ...string) *Route {
	return mux.route(http.MethodPut, path, handler, names...)
}

// Put registers a handler that can be used to serve any PUT
// request matching the given path pattern.
func (mux *ServeMux) Put(path string, handler http.HandlerFunc, names ...string) *Route {
	return mux.PutHandler(path, http.HandlerFunc(handler), names...)
}

// PatchHandler registers a handler that can be used to serve any PATCH
// request matching the given path pattern.
func (mux *ServeMux) PatchHandler(path string, handler http.Handler, names ...string) *Route {
	return mux.route(http.MethodPatch, path, handler, names...)
}

// Patch registers a handler that can be used to serve any PATCH
// request matching the given path pattern.
func (mux *ServeMux) Patch(path string, handler http.HandlerFunc, names ...string) *Route {
	return mux.PatchHandler(path, http.HandlerFunc(handler), names...)
}

// DeleteHandler registers a handler that can be used to serve any DELETE
// request matching the given path pattern.
func (mux *ServeMux) DeleteHandler(path string, handler http.Handler, names ...string) *Route {
	return mux.route(http.MethodDelete, path, handler, names...)
}

// Delete registers a handler that can be used to serve any DELETE
// request matching the given path pattern.
func (mux *ServeMux) Delete(path string, handler http.HandlerFunc, names ...string) *Route {
	return mux.DeleteHandler(path, http.HandlerFunc(handler), names...)
}

// AnyHandler registers a handler that can be used to serve any request matching
// the given path pattern.
func (mux *ServeMux) AnyHandler(path string, handler http.Handler, names ...string) *Route {
	route := mux.OptionsHandler(path, handler, names...)
	mux.ConnectHandler(path, handler)
	mux.TraceHandler(path, handler)
	mux.HeadHandler(path, handler)
	mux.GetHandler(path, handler)
	mux.PostHandler(path, handler)
	mux.PutHandler(path, handler)
	mux.PatchHandler(path, handler)
	mux.DeleteHandler(path, handler)

	return route
}

// Any registers a handler that can be used to serve any request matching
// the given path pattern.
func (mux *ServeMux) Any(path string, handler http.HandlerFunc, names ...string) *Route {
	return mux.AnyHandler(path, http.HandlerFunc(handler), names...)
}

// NotFoundHandler registers a handler to be used when an HTTP not found error
// is triggered.
func (mux *ServeMux) NotFoundHandler(handler http.Handler) {
	mux.notFound = handler
}

// NotFound registers a handler to be used when an HTTP not found error
// is triggered.
func (mux *ServeMux) NotFound(handler http.HandlerFunc) {
	mux.NotFoundHandler(http.HandlerFunc(handler))
}

// MethodNotAllowedHandler registers a handler to be used when an HTTP method
// not allowed error is triggered.
func (mux *ServeMux) MethodNotAllowedHandler(handler http.Handler) {
	mux.methodNotAllowed = handler
}

// MethodNotAllowed registers a handler to be used when an HTTP method
// not allowed error is triggered.
func (mux *ServeMux) MethodNotAllowed(handler http.HandlerFunc) {
	mux.MethodNotAllowedHandler(http.HandlerFunc(handler))
}

// Rewrite will register a new route that rewrites the source path pattern to
// the destination path pattern, and then attempts to find a handler that
// matches the new pattern instead.
//
// Any parameters used in the source path will have their value replaced into
// the destination path where the same parameter is used.
// For example, the patterns: "/:foo/greet"; "/:foo/world", will rewrite the
// source "/hello/greet" to "/hello/world".
func (mux *ServeMux) Rewrite(method, src, dst string) {
	originalPrefix := mux.prefix
	mux.prefix = ""

	mux.route(method, src, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

		mux.ServeHTTP(w, r)
	}))

	mux.prefix = originalPrefix
}

// Redirect will create a new handler for the given source path that will
// redirect the request to the given destination path using the code.
//
// Any parameters used in the source path will have their value replaced into
// the destination path where the same parameter is used.
// For example, the patterns: "/:foo/greet"; "/:foo/world", will redirect the
// source "/hello/greet" to "/hello/world".
func (mux *ServeMux) Redirect(method, src, dst string, code int) {
	originalPrefix := mux.prefix
	mux.prefix = ""

	mux.route(method, src, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

	mux.prefix = originalPrefix
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
		as.SetBool(str == "1" || str == "on")

	case reflect.Float32:
		var value float64
		if str != "" {
			value, err = strconv.ParseFloat(str, 32)
			if err != nil {
				return res, fmt.Errorf("parse float32: %w", err)
			}
		}

		as.SetFloat(value)

	case reflect.Float64:
		var value float64
		if str != "" {
			value, err = strconv.ParseFloat(str, 64)
			if err != nil {
				return res, fmt.Errorf("parse float64: %w", err)
			}
		}

		as.SetFloat(value)

	case reflect.Int8:
		var value int64
		if str != "" {
			value, err = strconv.ParseInt(str, 10, 8)
			if err != nil {
				return res, fmt.Errorf("parse int8: %w", err)
			}
		}

		as.SetInt(value)

	case reflect.Int16:
		var value int64
		if str != "" {
			value, err = strconv.ParseInt(str, 10, 16)
			if err != nil {
				return res, fmt.Errorf("parse int16: %w", err)
			}
		}

		as.SetInt(value)

	case reflect.Int32:
		var value int64
		if str != "" {
			value, err = strconv.ParseInt(str, 10, 32)
			if err != nil {
				return res, fmt.Errorf("parse int32: %w", err)
			}
		}

		as.SetInt(value)

	case reflect.Int64:
		var value int64
		if str != "" {
			value, err = strconv.ParseInt(str, 10, 64)
			if err != nil {
				return res, fmt.Errorf("parse int64: %w", err)
			}
		}

		as.SetInt(value)

	case reflect.Int:
		var value int64
		if str != "" {
			value, err = strconv.ParseInt(str, 10, 64)
			if err != nil {
				return res, fmt.Errorf("parse int: %w", err)
			}
		}

		as.SetInt(value)

	case reflect.Uint8:
		var value uint64
		if str != "" {
			value, err = strconv.ParseUint(str, 10, 8)
			if err != nil {
				return res, fmt.Errorf("parse uint8: %w", err)
			}
		}

		as.SetUint(value)

	case reflect.Uint16:
		var value uint64
		if str != "" {
			value, err = strconv.ParseUint(str, 10, 16)
			if err != nil {
				return res, fmt.Errorf("parse uint16: %w", err)
			}
		}

		as.SetUint(value)

	case reflect.Uint32:
		var value uint64
		if str != "" {
			value, err = strconv.ParseUint(str, 10, 32)
			if err != nil {
				return res, fmt.Errorf("parse uint32: %w", err)
			}
		}

		as.SetUint(value)

	case reflect.Uint64:
		var value uint64
		if str != "" {
			value, err = strconv.ParseUint(str, 10, 64)
			if err != nil {
				return res, fmt.Errorf("parse uint64: %w", err)
			}
		}

		as.SetUint(value)

	case reflect.Uint:
		var value uint64
		if str != "" {
			value, err = strconv.ParseUint(str, 10, 64)
			if err != nil {
				return res, fmt.Errorf("parse uint: %w", err)
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

	return res, nil
}
