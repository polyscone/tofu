package router_test

import (
	"io"
	"net/http"
	"testing"

	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/http/router"
	"github.com/polyscone/tofu/internal/pkg/testutil"
)

func TestMux(t *testing.T) {
	mux := router.NewServeMux()

	emptyHandler := func(w http.ResponseWriter, r *http.Request) {}

	mux.Prefix("/route", func(mux *router.ServeMux) {
		mux.Prefix("/test", func(mux *router.ServeMux) {
			mux.Options("", emptyHandler)
			mux.Connect("", emptyHandler)
			mux.Trace("", emptyHandler)
			mux.Head("", emptyHandler)
			mux.Get("", emptyHandler)
			mux.Post("", emptyHandler)
			mux.Put("", emptyHandler)
			mux.Patch("", emptyHandler)
			mux.Delete("", emptyHandler)
		})
	})

	mux.Prefix("/get", func(mux *router.ServeMux) {
		mux.Prefix("/only", func(mux *router.ServeMux) {
			mux.Get("", emptyHandler)
		})
	})

	mux.Prefix("/handle", func(mux *router.ServeMux) {
		mux.Prefix("/all", func(mux *router.ServeMux) {
			mux.Any("", emptyHandler)
		})
	})

	mux.Prefix("/same", func(mux *router.ServeMux) {
		mux.Prefix("/prefix", func(mux *router.ServeMux) {
			mux.Get("/foo/bar", func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusTooManyRequests)
			})

			mux.Get("/foo", func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusTeapot)
			})

			mux.Get("/foo/baz", func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusConflict)
			})
		})
	})

	mux.Get("/url/:ignore/:status/qux", func(w http.ResponseWriter, r *http.Request) {
		switch router.URLParam(r, "status") {
		case "teapot":
			w.WriteHeader(http.StatusTeapot)

		case "gone":
			w.WriteHeader(http.StatusGone)

		default:
			w.WriteHeader(http.StatusBadRequest)
		}
	})

	mux.Get("/cat/:first/dog/:rest", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(router.URLParam(r, "first") + "/" + router.URLParam(r, "rest")))
	})

	ts := testutil.NewServer(t, mux)
	defer ts.Close()

	tt := []struct {
		name       string
		method     string
		path       string
		wantBody   string
		wantStatus int
	}{
		{"options method ok", http.MethodOptions, "/route/test", "", http.StatusOK},
		{"connect method ok", http.MethodConnect, "/route/test", "", http.StatusOK},
		{"trace method ok", http.MethodTrace, "/route/test", "", http.StatusOK},
		{"head method ok", http.MethodHead, "/route/test", "", http.StatusOK},
		{"get method ok", http.MethodGet, "/route/test", "", http.StatusOK},
		{"post method ok", http.MethodPost, "/route/test", "", http.StatusOK},
		{"put method ok", http.MethodPut, "/route/test", "", http.StatusOK},
		{"patch method ok", http.MethodPatch, "/route/test", "", http.StatusOK},
		{"delete method ok", http.MethodDelete, "/route/test", "", http.StatusOK},

		{"options method not found", http.MethodOptions, "/not/found", "", http.StatusNotFound},
		{"connect method not found", http.MethodConnect, "/not/found", "", http.StatusNotFound},
		{"trace method not found", http.MethodTrace, "/not/found", "", http.StatusNotFound},
		{"head method not found", http.MethodHead, "/not/found", "", http.StatusNotFound},
		{"get method not found", http.MethodGet, "/not/found", "", http.StatusNotFound},
		{"post method not found", http.MethodPost, "/not/found", "", http.StatusNotFound},
		{"put method not found", http.MethodPut, "/not/found", "", http.StatusNotFound},
		{"patch method not found", http.MethodPatch, "/not/found", "", http.StatusNotFound},
		{"delete method not found", http.MethodDelete, "/not/found", "", http.StatusNotFound},

		{"get only options method no content", http.MethodOptions, "/get/only", "", http.StatusNoContent},
		{"get only connect method not allowed", http.MethodConnect, "/get/only", "", http.StatusMethodNotAllowed},
		{"get only trace method not allowed", http.MethodTrace, "/get/only", "", http.StatusMethodNotAllowed},
		{"get only head method not allowed", http.MethodHead, "/get/only", "", http.StatusMethodNotAllowed},
		{"get only get method ok", http.MethodGet, "/get/only", "", http.StatusOK},
		{"get only post method not allowed", http.MethodPost, "/get/only", "", http.StatusMethodNotAllowed},
		{"get only put method not allowed", http.MethodPut, "/get/only", "", http.StatusMethodNotAllowed},
		{"get only patch method not allowed", http.MethodPatch, "/get/only", "", http.StatusMethodNotAllowed},
		{"get only delete method not allowed", http.MethodDelete, "/get/only", "", http.StatusMethodNotAllowed},

		{"generic options method ok", http.MethodOptions, "/handle/all", "", http.StatusOK},
		{"generic connect method ok", http.MethodConnect, "/handle/all", "", http.StatusOK},
		{"generic trace method ok", http.MethodTrace, "/handle/all", "", http.StatusOK},
		{"generic head method ok", http.MethodHead, "/handle/all", "", http.StatusOK},
		{"generic get method ok", http.MethodGet, "/handle/all", "", http.StatusOK},
		{"generic post method ok", http.MethodPost, "/handle/all", "", http.StatusOK},
		{"generic put method ok", http.MethodPut, "/handle/all", "", http.StatusOK},
		{"generic patch method ok", http.MethodPatch, "/handle/all", "", http.StatusOK},
		{"generic delete method ok", http.MethodDelete, "/handle/all", "", http.StatusOK},

		{"same prefix teapot", http.MethodGet, "/same/prefix/foo", "", http.StatusTeapot},
		{"same prefix too many requests", http.MethodGet, "/same/prefix/foo/bar", "", http.StatusTooManyRequests},
		{"same prefix conflict", http.MethodGet, "/same/prefix/foo/baz", "", http.StatusConflict},

		{"dynamic url param teapot", http.MethodGet, "/url/param/teapot/qux", "", http.StatusTeapot},
		{"dynamic url param gone", http.MethodGet, "/url/param/gone/qux", "", http.StatusGone},
		{"dynamic url bad request", http.MethodGet, "/url/param/foo/qux", "", http.StatusBadRequest},
		{"dynamic url with rest", http.MethodGet, "/cat/foo/dog/baz/qux", "foo/baz/qux", http.StatusOK},
		{"dynamic url with empty rest", http.MethodGet, "/cat/foo/dog/", "foo/", http.StatusOK},
		{"dynamic url with rest no match", http.MethodGet, "/cat/foo/bar/dog/baz/qux", "", http.StatusNotFound},
	}
	for _, tc := range tt {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			req := errors.Must(http.NewRequest(tc.method, ts.URL+tc.path, nil))
			res := errors.Must(ts.Client().Do(req))

			defer res.Body.Close()

			if tc.wantBody != "" {
				body, err := io.ReadAll(res.Body)
				if err != nil {
					t.Errorf("want <nil>; got %q", err)
				}
				if want, got := tc.wantBody, string(body); want != got {
					t.Errorf("want %q; got %q", want, got)
				}
			}

			if want, got := tc.wantStatus, res.StatusCode; want != got {
				t.Errorf("want %v; got %v", want, got)
			}
		})
	}
}

func TestMuxPanics(t *testing.T) {
	emptyHandler := func(w http.ResponseWriter, r *http.Request) {}

	t.Run("panic on duplicate route paths", func(t *testing.T) {
		defer func() {
			if recover() == nil {
				t.Error("want panic; got <nil>")
			}
		}()

		mux := router.NewServeMux()

		mux.Get("/one/two/three/four", emptyHandler)
		mux.Get("/one/two/three/four", emptyHandler)
	})

	t.Run("no panic on duplicate route paths with different methods", func(t *testing.T) {
		defer func() {
			if recover() != nil {
				t.Error("want <nil>; got panic")
			}
		}()

		mux := router.NewServeMux()

		mux.Get("/one/two/three/four", emptyHandler)
		mux.Post("/one/two/three/four", emptyHandler)
	})

	t.Run("panic on duplicate route paths with parameters", func(t *testing.T) {
		defer func() {
			if recover() == nil {
				t.Error("want panic; got <nil>")
			}
		}()

		mux := router.NewServeMux()

		mux.Get("/one/two/:foo/four", emptyHandler)
		mux.Get("/one/two/:bar/four", emptyHandler)
	})

	t.Run("no panic on duplicate route paths with parameters and different methods", func(t *testing.T) {
		defer func() {
			if recover() != nil {
				t.Error("want <nil>; got panic")
			}
		}()

		mux := router.NewServeMux()

		mux.Get("/one/two/:foo/four", emptyHandler)
		mux.Post("/one/two/:bar/four", emptyHandler)
	})
}
