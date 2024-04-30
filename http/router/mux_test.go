package router_test

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/polyscone/tofu/errsx"
	"github.com/polyscone/tofu/http/router"
	"github.com/polyscone/tofu/testutil"
)

func TestMux2(t *testing.T) {
	mux := router.NewServeMux()

	ts := testutil.NewServer(t, mux)
	defer ts.Close()

	echoHandler := func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(r.URL.Path))
	}

	mux.Before(func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			r.URL.Path += "/bar/"
			r.URL.Path = strings.ReplaceAll(r.URL.Path, "//", "/")

			next(w, r)
		}
	})

	mux.Use(func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			r.URL.Path += "/foo/"
			r.URL.Path = strings.ReplaceAll(r.URL.Path, "//", "/")

			next(w, r)
		}
	})

	mux.HandleFunc("OPTIONS /foo/{$}", echoHandler)
	mux.HandleFunc("CONNECT /foo/{$}", echoHandler)
	mux.HandleFunc("TRACE /foo/{$}", echoHandler)
	mux.HandleFunc("HEAD /foo/{$}", echoHandler)
	mux.HandleFunc("GET /foo/{$}", echoHandler)
	mux.HandleFunc("POST /foo/{$}", echoHandler)
	mux.HandleFunc("PUT /foo/{$}", echoHandler)
	mux.HandleFunc("PATCH /foo/{$}", echoHandler)
	mux.HandleFunc("DELETE /foo/{$}", echoHandler)

	mux.Group(func(mux *router.ServeMux) {
		mux.Before(func(next http.HandlerFunc) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				r.URL.Path += "/qux/"
				r.URL.Path = strings.ReplaceAll(r.URL.Path, "//", "/")

				next(w, r)
			}
		})

		mux.HandleFunc("OPTIONS /baz/", echoHandler)
		mux.HandleFunc("CONNECT /baz/", echoHandler)
		mux.HandleFunc("TRACE /baz/", echoHandler)
		mux.HandleFunc("HEAD /baz/{$}", echoHandler)
		mux.HandleFunc("GET /baz/", echoHandler)
		mux.HandleFunc("POST /baz/", echoHandler)
		mux.HandleFunc("PUT /baz/", echoHandler)
		mux.HandleFunc("PATCH /baz/", echoHandler)
		mux.HandleFunc("DELETE /baz/", echoHandler)
	})

	mux.HandleFunc("OPTIONS /quxx/foo/", echoHandler)
	mux.HandleFunc("CONNECT /quxx/foo/", echoHandler)
	mux.HandleFunc("TRACE /quxx/foo/", echoHandler)
	mux.HandleFunc("HEAD /quxx/foo/", echoHandler)
	mux.HandleFunc("GET /quxx/foo/", echoHandler)
	mux.HandleFunc("POST /quxx/foo/", echoHandler)
	mux.HandleFunc("PUT /quxx/foo/", echoHandler)
	mux.HandleFunc("PATCH /quxx/foo/", echoHandler)
	mux.HandleFunc("DELETE /quxx/foo/", echoHandler)

	tt := []struct {
		name       string
		method     string
		path       string
		wantBody   string
		wantStatus int
	}{
		{"options method ok", http.MethodOptions, "/", "/foo/bar/", http.StatusOK},
		{"connect method ok", http.MethodConnect, "/", "/foo/bar/", http.StatusOK},
		{"trace method ok", http.MethodTrace, "/", "/foo/bar/", http.StatusOK},
		{"head method ok", http.MethodHead, "/", "", http.StatusOK},
		{"get method ok", http.MethodGet, "/", "/foo/bar/", http.StatusOK},
		{"post method ok", http.MethodPost, "/", "/foo/bar/", http.StatusOK},
		{"put method ok", http.MethodPut, "/", "/foo/bar/", http.StatusOK},
		{"patch method ok", http.MethodPatch, "/", "/foo/bar/", http.StatusOK},
		{"delete method ok", http.MethodDelete, "/", "/foo/bar/", http.StatusOK},

		{"options method ok in group", http.MethodOptions, "/baz/foo/", "/baz/foo/foo/bar/qux/", http.StatusOK},
		{"connect method ok in group", http.MethodConnect, "/baz/foo/", "/baz/foo/foo/bar/qux/", http.StatusOK},
		{"trace method ok in group", http.MethodTrace, "/baz/foo/", "/baz/foo/foo/bar/qux/", http.StatusOK},
		{"head method ok in group", http.MethodHead, "/baz/foo/", "", http.StatusOK},
		{"get method ok in group", http.MethodGet, "/baz/foo/", "/baz/foo/foo/bar/qux/", http.StatusOK},
		{"post method ok in group", http.MethodPost, "/baz/foo/", "/baz/foo/foo/bar/qux/", http.StatusOK},
		{"put method ok in group", http.MethodPut, "/baz/foo/", "/baz/foo/foo/bar/qux/", http.StatusOK},
		{"patch method ok in group", http.MethodPatch, "/baz/foo/", "/baz/foo/foo/bar/qux/", http.StatusOK},
		{"delete method ok in group", http.MethodDelete, "/baz/foo/", "/baz/foo/foo/bar/qux/", http.StatusOK},

		{"options method ok specific", http.MethodOptions, "/quxx", "/quxx/foo/bar/", http.StatusOK},
		{"connect method ok specific", http.MethodConnect, "/quxx", "/quxx/foo/bar/", http.StatusOK},
		{"trace method ok specific", http.MethodTrace, "/quxx", "/quxx/foo/bar/", http.StatusOK},
		{"head method ok specific", http.MethodHead, "/quxx", "", http.StatusOK},
		{"get method ok specific", http.MethodGet, "/quxx", "/quxx/foo/bar/", http.StatusOK},
		{"post method ok specific", http.MethodPost, "/quxx", "/quxx/foo/bar/", http.StatusOK},
		{"put method ok specific", http.MethodPut, "/quxx", "/quxx/foo/bar/", http.StatusOK},
		{"patch method ok specific", http.MethodPatch, "/quxx", "/quxx/foo/bar/", http.StatusOK},
		{"delete method ok specific", http.MethodDelete, "/quxx", "/quxx/foo/bar/", http.StatusOK},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			req := errsx.Must(http.NewRequest(tc.method, ts.URL+tc.path, nil))
			res := errsx.Must(ts.Client().Do(req))

			defer res.Body.Close()

			if tc.wantBody != "" {
				body, err := io.ReadAll(res.Body)
				if err != nil {
					t.Errorf("%v %v: want <nil>; got %q", tc.method, tc.path, err)
				}
				if want, got := tc.wantBody, string(body); want != got {
					t.Errorf("%v %v: want %q; got %q", tc.method, tc.path, want, got)
				}
			}

			if want, got := tc.wantStatus, res.StatusCode; want != got {
				t.Errorf("%v %v: want %v; got %v", tc.method, tc.path, want, got)
			}
		})
	}
}

func TestMuxNamedRoutes(t *testing.T) {
	mux := router.NewServeMux()

	emptyHandler := func(w http.ResponseWriter, r *http.Request) {}

	mux.HandleFunc("GET /named/route/here", emptyHandler, "named.route.1")

	if want, got := "/named/route/here", mux.Path("named.route.1"); want != got {
		t.Errorf("want path %q; got %q", want, got)
	}

	mux.HandleFunc("/named/route/here/2", emptyHandler, "named.route.2")

	if want, got := "/named/route/here/2", mux.Path("named.route.2"); want != got {
		t.Errorf("want path %q; got %q", want, got)
	}

	mux.HandleFunc("/named/route/{foo}/3/{bar...}", emptyHandler, "named.route.3")

	p := mux.Path("named.route.3", "{foo}", "hello", "{bar...}", "world/qux")
	if want, got := "/named/route/hello/3/world/qux", p; want != got {
		t.Errorf("want path %q; got %q", want, got)
	}
}
