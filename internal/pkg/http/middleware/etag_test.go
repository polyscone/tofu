package middleware_test

import (
	"net/http"
	"testing"

	"github.com/polyscone/tofu/internal/pkg/errsx"
	"github.com/polyscone/tofu/internal/pkg/http/middleware"
	"github.com/polyscone/tofu/internal/pkg/http/router"
	"github.com/polyscone/tofu/internal/pkg/testutil"
)

func TestETag(t *testing.T) {
	mux := router.NewServeMux()

	mux.Use(middleware.ETag)

	mux.Get("/hello/world", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello, World!"))
	})

	mux.Get("/foo/bar", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Foo bar baz"))
	})

	mux.Get("/no/content", func(w http.ResponseWriter, r *http.Request) {})

	ts := testutil.NewServer(t, mux)
	defer ts.Close()

	tt := []struct {
		name   string
		method string
		path   string
		match  string
		want   string
	}{
		{"etag for hello path", http.MethodGet, "/hello/world", "", "65a8e27d8879283831b664bd8b7f0ad4"},
		{"etag for foo path", http.MethodGet, "/foo/bar", "", "520c28a8ac3459af817a1abfb3bd152e"},
		{"no etag in no content", http.MethodGet, "/no/content", "", ""},
		{"304 not modified for matching if-none-match", http.MethodGet, "/hello/world", "65a8e27d8879283831b664bd8b7f0ad4", "65a8e27d8879283831b664bd8b7f0ad4"},
		{"200 ok for different if-none-match", http.MethodGet, "/hello/world", "123", "65a8e27d8879283831b664bd8b7f0ad4"},
		{"no etag for head", http.MethodHead, "/hello/world", "", ""},
		{"no etag for post", http.MethodPost, "/hello/world", "", ""},
		{"no etag for put", http.MethodPut, "/hello/world", "", ""},
		{"no etag for delete", http.MethodDelete, "/hello/world", "", ""},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			req := errsx.Must(http.NewRequest(tc.method, ts.URL+tc.path, nil))

			if tc.match != "" {
				req.Header.Set("if-none-match", tc.match)
			}

			res := errsx.Must(ts.Client().Do(req))

			defer res.Body.Close()

			if want, got := tc.want, res.Header.Get("etag"); want != got {
				t.Errorf("want %q; got %q", want, got)
			}

			if tc.match != "" {
				if tc.match == tc.want {
					if want, got := http.StatusNotModified, res.StatusCode; want != got {
						t.Errorf("want %v; got %v", want, got)
					}
				}

				if tc.match != tc.want {
					if want, got := http.StatusOK, res.StatusCode; want != got {
						t.Errorf("want %v; got %v", want, got)
					}
				}
			}
		})
	}
}
