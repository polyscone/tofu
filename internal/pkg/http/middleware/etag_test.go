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

	mux.Use(middleware.ETag(nil))

	mux.HandleFunc("GET /hello/world", func(w http.ResponseWriter, r *http.Request) {
		if etags := r.URL.Query()["set-etag"]; len(etags) != 0 {
			w.Header().Set("etag", etags[0])
		}

		w.Write([]byte("Hello, World!"))
	})

	mux.HandleFunc("GET /foo/bar", func(w http.ResponseWriter, r *http.Request) {
		if etags := r.URL.Query()["set-etag"]; len(etags) != 0 {
			w.Header().Set("etag", etags[0])
		}

		w.Write([]byte("Foo bar baz"))
	})

	mux.HandleFunc("GET /no/content", func(w http.ResponseWriter, r *http.Request) {
		if etags := r.URL.Query()["set-etag"]; len(etags) != 0 {
			w.Header().Set("etag", etags[0])
		}
	})

	ts := testutil.NewServer(t, mux)
	defer ts.Close()

	tt := []struct {
		name   string
		method string
		path   string
		etag   string
		match  string
		want   string
	}{
		{"etag for hello path", http.MethodGet, "/hello/world", "", "", "65a8e27d8879283831b664bd8b7f0ad4"},
		{"etag for foo path", http.MethodGet, "/foo/bar", "", "", "520c28a8ac3459af817a1abfb3bd152e"},
		{"etag for foo path populated override", http.MethodGet, "/foo/bar", "123abc", "", "123abc"},
		{"etag for foo path empty orderride", http.MethodGet, "/foo/bar", "<empty>", "", ""},
		{"no etag in no content", http.MethodGet, "/no/content", "", "", ""},
		{"304 not modified for matching if-none-match", http.MethodGet, "/hello/world", "", "65a8e27d8879283831b664bd8b7f0ad4", "65a8e27d8879283831b664bd8b7f0ad4"},
		{"200 ok for different if-none-match", http.MethodGet, "/hello/world", "", "123", "65a8e27d8879283831b664bd8b7f0ad4"},
		{"no etag for head", http.MethodHead, "/hello/world", "", "", ""},
		{"no etag for post", http.MethodPost, "/hello/world", "", "", ""},
		{"no etag for put", http.MethodPut, "/hello/world", "", "", ""},
		{"no etag for delete", http.MethodDelete, "/hello/world", "", "", ""},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			req := errsx.Must(http.NewRequest(tc.method, ts.URL+tc.path, nil))

			if tc.match != "" {
				req.Header.Set("if-none-match", tc.match)
			}
			if tc.etag != "" {
				q := req.URL.Query()

				if tc.etag == "<empty>" {
					tc.etag = ""
				}

				q.Set("set-etag", tc.etag)

				req.URL.RawQuery = q.Encode()
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
