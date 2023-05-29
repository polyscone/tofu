package middleware_test

import (
	"io"
	"net/http"
	"testing"

	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/http/middleware"
	"github.com/polyscone/tofu/internal/pkg/http/router"
	"github.com/polyscone/tofu/internal/pkg/testutil"
)

func TestRemoveTrailingSlash(t *testing.T) {
	mux := router.NewServeMux()

	mux.Use(middleware.RemoveTrailingSlash)

	mux.Get("/", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(r.URL.String())) })
	mux.Post("/", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(r.URL.String())) })
	mux.Put("/", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(r.URL.String())) })
	mux.Patch("/", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(r.URL.String())) })
	mux.Delete("/", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(r.URL.String())) })

	mux.Get("/foo", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(r.URL.String())) })
	mux.Post("/foo", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(r.URL.String())) })
	mux.Put("/foo", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(r.URL.String())) })
	mux.Patch("/foo", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(r.URL.String())) })
	mux.Delete("/foo", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(r.URL.String())) })

	mux.Get("/foo/", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(r.URL.String())) })
	mux.Post("/foo/", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(r.URL.String())) })
	mux.Put("/foo/", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(r.URL.String())) })
	mux.Patch("/foo/", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(r.URL.String())) })
	mux.Delete("/foo/", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(r.URL.String())) })

	ts := testutil.NewServer(t, mux)
	defer ts.Close()

	tt := []struct {
		name   string
		method string
		path   string
		want   string
	}{
		{"get root slash", http.MethodGet, "/", "/"},
		{"post root slash", http.MethodPost, "/", "/"},
		{"put root slash", http.MethodPut, "/", "/"},
		{"patch root slash", http.MethodPatch, "/", "/"},
		{"delete root slash", http.MethodDelete, "/", "/"},

		{"get no trailing slash", http.MethodGet, "/foo", "/foo"},
		{"post no trailing slash", http.MethodPost, "/foo", "/foo"},
		{"put no trailing slash", http.MethodPut, "/foo", "/foo"},
		{"patch no trailing slash", http.MethodPatch, "/foo", "/foo"},
		{"delete no trailing slash", http.MethodDelete, "/foo", "/foo"},

		{"get trailing slash", http.MethodGet, "/foo/", "/foo"},
		{"post trailing slash", http.MethodPost, "/foo/", "/foo"},
		{"put trailing slash", http.MethodPut, "/foo/", "/foo"},
		{"patch trailing slash", http.MethodPatch, "/foo/", "/foo"},
		{"delete trailing slash", http.MethodDelete, "/foo/", "/foo"},

		{"get trailing slash with query", http.MethodGet, "/foo/?bar=baz&qux=quxx", "/foo?bar=baz&qux=quxx"},
		{"post trailing slash with query", http.MethodPost, "/foo/?bar=baz&qux=quxx", "/foo?bar=baz&qux=quxx"},
		{"put trailing slash with query", http.MethodPut, "/foo/?bar=baz&qux=quxx", "/foo?bar=baz&qux=quxx"},
		{"patch trailing slash with query", http.MethodPatch, "/foo/?bar=baz&qux=quxx", "/foo?bar=baz&qux=quxx"},
		{"delete trailing slash with query", http.MethodDelete, "/foo/?bar=baz&qux=quxx", "/foo?bar=baz&qux=quxx"},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			req := errors.Must(http.NewRequest(tc.method, ts.URL+tc.path, nil))

			req.Header.Set("content-type", "application/x-www-form-urlencoded")

			res := errors.Must(ts.Client().Do(req))

			defer res.Body.Close()

			got := string(errors.Must(io.ReadAll(res.Body)))
			if want := tc.want; want != got {
				t.Errorf("want %v; got %v", want, got)
			}
		})
	}
}
