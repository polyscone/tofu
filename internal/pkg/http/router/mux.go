package router

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strings"

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
	handlers map[string]http.Handler
	methods  []string
}

// ServeMux represents an HTTP router.
type ServeMux struct {
	prefix           string
	middlewares      []middleware.Middleware
	routes           []Route
	notFound         http.Handler
	methodNotAllowed http.Handler
}

// NewServeMux returns a new serve mux.
func NewServeMux() *ServeMux {
	return &ServeMux{}
}

// ServeHTTP implements the http.Handler interface.
func (sm *ServeMux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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

// Use adds a middleware function to the middleware stack to be called
// before any handlers.
// Middleware registered with this function are called in the order they
// are registered.
func (sm *ServeMux) Use(middleware middleware.Middleware) {
	sm.middlewares = append(sm.middlewares, middleware)
}

// Prefix will automatically prefix any path patterns that are registered in
// given the route group function with the given prefix.
func (sm *ServeMux) Prefix(prefix string, routeGroup func(*ServeMux)) {
	originalPrefix := sm.prefix
	sm.prefix += prefix

	routeGroup(sm)

	sm.prefix = originalPrefix
}

func (sm *ServeMux) route(method string, path string, handler http.Handler) {
	method = strings.ToUpper(method)
	path = sm.prefix + path
	pattern := regexp.QuoteMeta(path)

	if pattern == "" {
		panic("route must not be empty")
	}

	pattern = reLastParam.ReplaceAllString(pattern, `(?P<$1>.*?)`)
	pattern = reParams.ReplaceAllString(pattern, `(?P<$1>[^/]+?)`)
	pattern = "^" + pattern + "$"

	key := reParams.ReplaceAllString(path, "*")

	handler = middleware.Apply(handler, sm.middlewares...)

	for _, route := range sm.routes {
		if route.key != key {
			continue
		}

		if _, ok := route.handlers[method]; ok {
			panic(fmt.Sprintf("duplicate route registration for %q and %q (%v)", path, route.path, method))
		}

		route.handlers[method] = handler
		route.methods = append(route.methods, method)

		return
	}

	compiled := regexp.MustCompile(pattern)

	seen := make(map[string]struct{})
	for _, name := range compiled.SubexpNames() {
		if _, ok := seen[name]; ok {
			panic(fmt.Sprintf("duplicate parameter name %q in route %q", name, path))
		}

		seen[name] = struct{}{}
	}

	sm.routes = append(sm.routes, Route{
		key:      key,
		path:     path,
		pattern:  compiled,
		handlers: map[string]http.Handler{method: handler},
		methods:  []string{http.MethodOptions},
	})
}

// OptionsHandler registers a handler that can be used to serve any OPTIONS
// request matching the given path pattern.
func (sm *ServeMux) OptionsHandler(path string, handler http.Handler) {
	sm.route(http.MethodOptions, path, handler)
}

// Options registers a handler that can be used to serve any OPTIONS
// request matching the given path pattern.
func (sm *ServeMux) Options(path string, handler http.HandlerFunc) {
	sm.OptionsHandler(path, http.HandlerFunc(handler))
}

// ConnectHandler registers a handler that can be used to serve any CONNECT
// request matching the given path pattern.
func (sm *ServeMux) ConnectHandler(path string, handler http.Handler) {
	sm.route(http.MethodConnect, path, handler)
}

// Connect registers a handler that can be used to serve any CONNECT
// request matching the given path pattern.
func (sm *ServeMux) Connect(path string, handler http.HandlerFunc) {
	sm.ConnectHandler(path, http.HandlerFunc(handler))
}

// TraceHandler registers a handler that can be used to serve any TRACE
// request matching the given path pattern.
func (sm *ServeMux) TraceHandler(path string, handler http.Handler) {
	sm.route(http.MethodTrace, path, handler)
}

// Trace registers a handler that can be used to serve any TRACE
// request matching the given path pattern.
func (sm *ServeMux) Trace(path string, handler http.HandlerFunc) {
	sm.TraceHandler(path, http.HandlerFunc(handler))
}

// HeadHandler registers a handler that can be used to serve any HEAD
// request matching the given path pattern.
func (sm *ServeMux) HeadHandler(path string, handler http.Handler) {
	sm.route(http.MethodHead, path, handler)
}

// Head registers a handler that can be used to serve any HEAD
// request matching the given path pattern.
func (sm *ServeMux) Head(path string, handler http.HandlerFunc) {
	sm.HeadHandler(path, http.HandlerFunc(handler))
}

// GetHandler registers a handler that can be used to serve any GET
// request matching the given path pattern.
func (sm *ServeMux) GetHandler(path string, handler http.Handler) {
	sm.route(http.MethodGet, path, handler)
}

// Get registers a handler that can be used to serve any GET
// request matching the given path pattern.
func (sm *ServeMux) Get(path string, handler http.HandlerFunc) {
	sm.GetHandler(path, http.HandlerFunc(handler))
}

// PostHandler registers a handler that can be used to serve any POST
// request matching the given path pattern.
func (sm *ServeMux) PostHandler(path string, handler http.Handler) {
	sm.route(http.MethodPost, path, handler)
}

// Post registers a handler that can be used to serve any POST
// request matching the given path pattern.
func (sm *ServeMux) Post(path string, handler http.HandlerFunc) {
	sm.PostHandler(path, http.HandlerFunc(handler))
}

// PutHandler registers a handler that can be used to serve any PUT
// request matching the given path pattern.
func (sm *ServeMux) PutHandler(path string, handler http.Handler) {
	sm.route(http.MethodPut, path, handler)
}

// Put registers a handler that can be used to serve any PUT
// request matching the given path pattern.
func (sm *ServeMux) Put(path string, handler http.HandlerFunc) {
	sm.PutHandler(path, http.HandlerFunc(handler))
}

// PatchHandler registers a handler that can be used to serve any PATCH
// request matching the given path pattern.
func (sm *ServeMux) PatchHandler(path string, handler http.Handler) {
	sm.route(http.MethodPatch, path, handler)
}

// Patch registers a handler that can be used to serve any PATCH
// request matching the given path pattern.
func (sm *ServeMux) Patch(path string, handler http.HandlerFunc) {
	sm.PatchHandler(path, http.HandlerFunc(handler))
}

// DeleteHandler registers a handler that can be used to serve any DELETE
// request matching the given path pattern.
func (sm *ServeMux) DeleteHandler(path string, handler http.Handler) {
	sm.route(http.MethodDelete, path, handler)
}

// Delete registers a handler that can be used to serve any DELETE
// request matching the given path pattern.
func (sm *ServeMux) Delete(path string, handler http.HandlerFunc) {
	sm.DeleteHandler(path, http.HandlerFunc(handler))
}

// AnyHandler registers a handler that can be used to serve any request matching
// the given path pattern.
func (sm *ServeMux) AnyHandler(path string, handler http.Handler) {
	sm.OptionsHandler(path, handler)
	sm.ConnectHandler(path, handler)
	sm.TraceHandler(path, handler)
	sm.HeadHandler(path, handler)
	sm.GetHandler(path, handler)
	sm.PostHandler(path, handler)
	sm.PutHandler(path, handler)
	sm.PatchHandler(path, handler)
	sm.DeleteHandler(path, handler)
}

// Any registers a handler that can be used to serve any request matching
// the given path pattern.
func (sm *ServeMux) Any(path string, handler http.HandlerFunc) {
	sm.AnyHandler(path, http.HandlerFunc(handler))
}

// NotFoundHandler registers a handler to be used when an HTTP not found error
// is triggered.
func (sm *ServeMux) NotFoundHandler(handler http.Handler) {
	sm.notFound = middleware.Apply(handler, sm.middlewares...)
}

// NotFound registers a handler to be used when an HTTP not found error
// is triggered.
func (sm *ServeMux) NotFound(handler http.HandlerFunc) {
	sm.NotFoundHandler(http.HandlerFunc(handler))
}

// MethodNotAllowedHandler registers a handler to be used when an HTTP method
// not allowed error is triggered.
func (sm *ServeMux) MethodNotAllowedHandler(handler http.Handler) {
	sm.methodNotAllowed = middleware.Apply(handler, sm.middlewares...)
}

// MethodNotAllowed registers a handler to be used when an HTTP method
// not allowed error is triggered.
func (sm *ServeMux) MethodNotAllowed(handler http.HandlerFunc) {
	sm.MethodNotAllowedHandler(http.HandlerFunc(handler))
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
