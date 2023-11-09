package router

import (
	"cmp"
	"context"
	"fmt"
	"maps"
	"net/http"
	"reflect"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/polyscone/tofu/internal/pkg/http/middleware"
)

type ctxKey int

const ctxParams ctxKey = iota

// Route represents a registered route and handler.
type Route struct {
	pattern  string
	parts    []string
	handlers map[string]http.Handler
	methods  []string
}

func (rt *Route) String() string {
	return rt.pattern
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
func (rt *Route) Replace(paramArgPairs ...any) string {
	if len(paramArgPairs)%2 == 1 {
		panic("route pattern substitution expects an equal number of arguments")
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

	if rt.parts == nil {
		rt.parts = strings.Split(strings.TrimPrefix(rt.pattern, "/"), "/")
	}

	seen := make(map[string]struct{})
	for _, part := range rt.parts {
		sb.WriteRune('/')

		if strings.HasPrefix(part, ":") {
			arg, ok := args[part]
			if !ok {
				panic(fmt.Sprintf("want an argument for parameter %q in route pattern to be provided", part))
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

func (rt *Route) handle(r *http.Request, w http.ResponseWriter, params map[string]string, methodNotAllowed http.Handler) {
	handler := rt.handlers[r.Method]
	if handler == nil {
		methods := strings.Join(rt.methods, ", ")
		if !slices.Contains(rt.methods, http.MethodOptions) {
			methods += ", " + http.MethodOptions
		}

		w.Header().Set("allow", methods)

		switch {
		case r.Method == http.MethodOptions:
			w.WriteHeader(http.StatusNoContent)

		case methodNotAllowed != nil:
			methodNotAllowed.ServeHTTP(w, r)

		default:
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		}

		return
	}

	if params != nil {
		ctx := context.WithValue(r.Context(), ctxParams, params)
		r = r.WithContext(ctx)
	}

	handler.ServeHTTP(w, r)
}

type Node struct {
	nodes map[string]*Node
	route *Route
}

// ServeMux represents an HTTP router.
type ServeMux struct {
	prefix           string
	middlewares      []middleware.Middleware
	befores          []middleware.Middleware
	handler          http.Handler
	static           map[string]*Node
	dynamic          map[string]*Node
	named            map[string]*Route
	notFound         http.Handler
	methodNotAllowed http.Handler
}

// NewServeMux returns a new serve mux.
func NewServeMux() *ServeMux {
	var mux ServeMux

	mux.handler = http.HandlerFunc(mux.serveHTTP)
	mux.static = make(map[string]*Node)
	mux.dynamic = make(map[string]*Node)

	return &mux
}

func (mux *ServeMux) serveHTTP(w http.ResponseWriter, r *http.Request) {
	var params map[string]string

	node := mux.static[r.URL.Path]
	if node == nil {
		params = make(map[string]string)
		dynamic := mux.dynamic
		parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/"), "/")
		last := len(parts) - 1

		// GreedyMatch represents a match for greedy patterns that end with "..."
		// The latest one found in the chain is stored as a fallback in case we
		// can't find a more specific node
		type GreedyMatch struct {
			node   *Node
			params map[string]string
		}

		var greedy *GreedyMatch
		for i, part := range parts {
			// First check to see if we have any mappings from a greedy pattern ending in "..."
			// If we do then we want to store it for later in case we need to use it as a fallback
			for key, n := range dynamic {
				if strings.HasPrefix(key, ":") && strings.HasSuffix(key, "...") {
					name := strings.TrimSuffix(key[1:], "...")
					params[name] = strings.Join(parts[i:], "/")

					greedy = &GreedyMatch{
						node:   n,
						params: maps.Clone(params),
					}

					// Each node's mappings to further nodes in the chain should only
					// contain at most one greedy pattern, so when we find it we can
					// just break out early
					break
				}
			}

			dynode := dynamic[part]
			if dynode == nil {
				// If we didn't find a static mapping to another node we try looking
				// for a dynamic mapping with a lazy pattern that doesn't end in "..."
				for key, n := range dynamic {
					if strings.HasPrefix(key, ":") && !strings.HasSuffix(key, "...") {
						dynode = n
						params[key[1:]] = part

						// Each node's mappings to further nodes in the chain should only
						// contain at most one lazy pattern, so when we find it we can
						// just break out early
						break
					}
				}
			}
			if dynode == nil {
				// If we didn't find a mapping to another node we give up here
				break
			}

			if i == last {
				node = dynode
			} else {
				dynamic = dynode.nodes
			}
		}

		// If we only found a greedy node we use that as a fallback
		if (node == nil || node.route == nil) && greedy != nil {
			node = greedy.node
			params = greedy.params
		}
	}

	if node != nil && node.route != nil {
		node.route.handle(r, w, params, mux.methodNotAllowed)

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

func (mux *ServeMux) Before(mw middleware.Middleware) {
	mux.befores = append(mux.befores, mw)
}

// Prefix will automatically prefix any patterns that are registered in
// given the route group function with the given prefix.
func (mux *ServeMux) Prefix(prefix string, routeGroup func(mux *ServeMux)) {
	originalPrefix := mux.prefix
	originalBefores := slices.Clone(mux.befores)

	mux.prefix += prefix
	if mux.prefix != "/" {
		mux.prefix = strings.TrimSuffix(mux.prefix, "/")
	}

	routeGroup(mux)

	mux.befores = originalBefores
	mux.prefix = originalPrefix
}

func (mux *ServeMux) CurrentPrefix() string {
	return mux.prefix + "/"
}

func (mux *ServeMux) CurrentPattern() string {
	return mux.prefix
}

func (mux *ServeMux) Name(name string) {
	route := &Route{pattern: mux.prefix}

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

// node creates a chain of node objects and returns the final one for the given pattern.
func (mux *ServeMux) node(pattern string) *Node {
	var node *Node
	if strings.Contains(pattern, "/:") {
		dynamic := mux.dynamic
		parts := strings.Split(strings.TrimPrefix(pattern, "/"), "/")
		last := len(parts) - 1

		for i, part := range parts {
			if strings.HasPrefix(part, ":") {
				isGreedy := strings.HasSuffix(part, "...")

				for key := range dynamic {
					if key == part || !strings.HasPrefix(key, ":") {
						continue
					}

					keyIsGreedy := strings.HasSuffix(key, "...")

					if isGreedy && keyIsGreedy || !isGreedy && !keyIsGreedy {
						panic(fmt.Sprintf("multiple parameters in the same position for %v", pattern))
					}
				}
			}

			if dynamic[part] == nil {
				node = &Node{}

				dynamic[part] = node
			}

			node = dynamic[part]

			if i != last {
				if node.nodes == nil {
					node.nodes = make(map[string]*Node)
				}

				dynamic = node.nodes
			}
		}
	} else {
		node = mux.static[pattern]
		if node == nil {
			node = &Node{}

			mux.static[pattern] = node
		}
	}

	return node
}

var reMultiSlash = regexp.MustCompile(`//+`)

func (mux *ServeMux) route(method, pattern string, handler http.Handler, names ...string) *Route {
	method = strings.ToUpper(method)

	if pattern == "" {
		panic("route pattern must not be empty")
	}

	if mux.prefix != "" && pattern == "/" {
		pattern = ""
	}

	pattern = mux.prefix + pattern
	pattern = reMultiSlash.ReplaceAllString(pattern, "/")

	if pattern == "" {
		panic("route must not be empty")
	}

	if strings.Contains(pattern, ".../") {
		panic("greedy ... syntax can only appear after the final named parameter")
	}

	node := mux.node(pattern)
	if node.route == nil {
		node.route = &Route{handlers: make(map[string]http.Handler)}
	}

	route := node.route
	if _, ok := route.handlers[method]; ok {
		panic(fmt.Sprintf("duplicate routes for %v %v", method, pattern))
	}

	if len(mux.befores) > 0 {
		handler = middleware.Apply(handler, mux.befores...)
	}

	route.pattern = pattern
	route.handlers[method] = handler
	route.methods = append(route.methods, method)

	mux.nameRoute(route, names...)

	return route
}

func (mux *ServeMux) HandleFunc(pattern string, handler http.HandlerFunc, names ...string) {

}

// OptionsHandler registers a handler that can be used to serve any OPTIONS
// request matching the given pattern.
func (mux *ServeMux) OptionsHandler(pattern string, handler http.Handler, names ...string) *Route {
	return mux.route(http.MethodOptions, pattern, handler, names...)
}

// Options registers a handler that can be used to serve any OPTIONS
// request matching the given pattern.
func (mux *ServeMux) Options(pattern string, handler http.HandlerFunc, names ...string) *Route {
	return mux.OptionsHandler(pattern, http.HandlerFunc(handler), names...)
}

// ConnectHandler registers a handler that can be used to serve any CONNECT
// request matching the given pattern.
func (mux *ServeMux) ConnectHandler(pattern string, handler http.Handler, names ...string) *Route {
	return mux.route(http.MethodConnect, pattern, handler, names...)
}

// Connect registers a handler that can be used to serve any CONNECT
// request matching the given pattern.
func (mux *ServeMux) Connect(pattern string, handler http.HandlerFunc, names ...string) *Route {
	return mux.ConnectHandler(pattern, http.HandlerFunc(handler), names...)
}

// TraceHandler registers a handler that can be used to serve any TRACE
// request matching the given pattern.
func (mux *ServeMux) TraceHandler(pattern string, handler http.Handler, names ...string) *Route {
	return mux.route(http.MethodTrace, pattern, handler, names...)
}

// Trace registers a handler that can be used to serve any TRACE
// request matching the given pattern.
func (mux *ServeMux) Trace(pattern string, handler http.HandlerFunc, names ...string) *Route {
	return mux.TraceHandler(pattern, http.HandlerFunc(handler), names...)
}

// HeadHandler registers a handler that can be used to serve any HEAD
// request matching the given pattern.
func (mux *ServeMux) HeadHandler(pattern string, handler http.Handler, names ...string) *Route {
	return mux.route(http.MethodHead, pattern, handler, names...)
}

// Head registers a handler that can be used to serve any HEAD
// request matching the given pattern.
func (mux *ServeMux) Head(pattern string, handler http.HandlerFunc, names ...string) *Route {
	return mux.HeadHandler(pattern, http.HandlerFunc(handler), names...)
}

// GetHandler registers a handler that can be used to serve any GET
// request matching the given pattern.
func (mux *ServeMux) GetHandler(pattern string, handler http.Handler, names ...string) *Route {
	return mux.route(http.MethodGet, pattern, handler, names...)
}

// Get registers a handler that can be used to serve any GET
// request matching the given pattern.
func (mux *ServeMux) Get(pattern string, handler http.HandlerFunc, names ...string) *Route {
	return mux.GetHandler(pattern, http.HandlerFunc(handler), names...)
}

// PostHandler registers a handler that can be used to serve any POST
// request matching the given pattern.
func (mux *ServeMux) PostHandler(pattern string, handler http.Handler, names ...string) *Route {
	return mux.route(http.MethodPost, pattern, handler, names...)
}

// Post registers a handler that can be used to serve any POST
// request matching the given pattern.
func (mux *ServeMux) Post(pattern string, handler http.HandlerFunc, names ...string) *Route {
	return mux.PostHandler(pattern, http.HandlerFunc(handler), names...)
}

// PutHandler registers a handler that can be used to serve any PUT
// request matching the given pattern.
func (mux *ServeMux) PutHandler(pattern string, handler http.Handler, names ...string) *Route {
	return mux.route(http.MethodPut, pattern, handler, names...)
}

// Put registers a handler that can be used to serve any PUT
// request matching the given pattern.
func (mux *ServeMux) Put(pattern string, handler http.HandlerFunc, names ...string) *Route {
	return mux.PutHandler(pattern, http.HandlerFunc(handler), names...)
}

// PatchHandler registers a handler that can be used to serve any PATCH
// request matching the given pattern.
func (mux *ServeMux) PatchHandler(pattern string, handler http.Handler, names ...string) *Route {
	return mux.route(http.MethodPatch, pattern, handler, names...)
}

// Patch registers a handler that can be used to serve any PATCH
// request matching the given pattern.
func (mux *ServeMux) Patch(pattern string, handler http.HandlerFunc, names ...string) *Route {
	return mux.PatchHandler(pattern, http.HandlerFunc(handler), names...)
}

// DeleteHandler registers a handler that can be used to serve any DELETE
// request matching the given pattern.
func (mux *ServeMux) DeleteHandler(pattern string, handler http.Handler, names ...string) *Route {
	return mux.route(http.MethodDelete, pattern, handler, names...)
}

// Delete registers a handler that can be used to serve any DELETE
// request matching the given pattern.
func (mux *ServeMux) Delete(pattern string, handler http.HandlerFunc, names ...string) *Route {
	return mux.DeleteHandler(pattern, http.HandlerFunc(handler), names...)
}

// AnyHandler registers a handler that can be used to serve any request matching
// the given pattern.
func (mux *ServeMux) AnyHandler(pattern string, handler http.Handler, names ...string) *Route {
	route := mux.OptionsHandler(pattern, handler, names...)
	mux.ConnectHandler(pattern, handler)
	mux.TraceHandler(pattern, handler)
	mux.HeadHandler(pattern, handler)
	mux.GetHandler(pattern, handler)
	mux.PostHandler(pattern, handler)
	mux.PutHandler(pattern, handler)
	mux.PatchHandler(pattern, handler)
	mux.DeleteHandler(pattern, handler)

	return route
}

// Any registers a handler that can be used to serve any request matching
// the given pattern.
func (mux *ServeMux) Any(pattern string, handler http.HandlerFunc, names ...string) *Route {
	return mux.AnyHandler(pattern, http.HandlerFunc(handler), names...)
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

// Rewrite will register a new route that rewrites the source pattern to
// the destination pattern, and then attempts to find a handler that
// matches the new pattern instead.
//
// Any parameters used in the source pattern will have their value replaced into
// the destination pattern where the same parameter is used.
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

			slices.SortFunc(keys, func(a, b string) int {
				// Reverse string length sort so the longest key comes first
				return cmp.Compare(utf8.RuneCountInString(b), utf8.RuneCountInString(a))
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

// Redirect will create a new handler for the given source pattern that will
// redirect the request to the given destination pattern using the code.
//
// Any parameters used in the source pattern will have their value replaced into
// the destination pattern where the same parameter is used.
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

			slices.SortFunc(keys, func(a, b string) int {
				// Reverse string length sort so the longest key comes first
				return cmp.Compare(utf8.RuneCountInString(b), utf8.RuneCountInString(a))
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
func URLParam(r *http.Request, name string) (string, bool) {
	value, ok := r.Context().Value(ctxParams).(map[string]string)[name]
	if !ok {
		return "", false
	}

	return value, true
}

// URLParamAs returns the value associated with the given parameter name in
// the given request URL after attempting to convert it to the given type T.
func URLParamAs[T any](r *http.Request, name string) (T, bool) {
	var res T

	str, ok := URLParam(r, name)
	if !ok {
		return res, false
	}

	var err error
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
