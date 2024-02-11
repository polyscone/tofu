package router_test

import (
	"io"
	"net/http"
	"testing"

	"github.com/polyscone/tofu/internal/pkg/errsx"
	"github.com/polyscone/tofu/internal/pkg/http/router"
	"github.com/polyscone/tofu/internal/pkg/testutil"
)

func TestMux(t *testing.T) {
	mux := router.NewServeMux()

	ts := testutil.NewServer(t, mux)
	defer ts.Close()

	emptyHandler := func(w http.ResponseWriter, r *http.Request) {}
	echoHandler := func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(r.URL.Path))
	}

	mux.Get("/order/{rest...}", echoHandler)
	mux.Get("/order/{foo}", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("lazy param: " + r.URL.Path))
	})
	mux.Get("/order/static", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("static: " + r.URL.Path))
	})
	mux.Get("/order/independent/{rest...}", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("greedy rest: " + r.URL.Path))
	})

	mux.Get("/order/independent/long/{rest...}", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("long greedy rest: " + r.URL.Path))
	})

	mux.Get("/order/independent/foo", echoHandler)
	mux.Get("/order/independent/foo/bar", echoHandler)

	mux.Options("/", echoHandler)
	mux.Connect("/", echoHandler)
	mux.Trace("/", echoHandler)
	mux.Head("/", echoHandler)
	mux.Get("/", echoHandler)
	mux.Post("/", echoHandler)
	mux.Put("/", echoHandler)
	mux.Patch("/", echoHandler)
	mux.Delete("/", echoHandler)

	mux.Group("/", func(mux *router.ServeMux) {
		mux.Get("/consecutive-slashes", emptyHandler)

		mux.Group("/route", func(mux *router.ServeMux) {

			mux.Group("/test", func(mux *router.ServeMux) {
				mux.Options("/", emptyHandler)
				mux.Connect("/", emptyHandler)
				mux.Trace("/", emptyHandler)
				mux.Head("/", emptyHandler)
				mux.Get("/", emptyHandler)
				mux.Post("/", emptyHandler)
				mux.Put("/", emptyHandler)
				mux.Patch("/", emptyHandler)
				mux.Delete("/", emptyHandler)
			})
		})
	})

	mux.Group("/get", func(mux *router.ServeMux) {
		mux.Group("/only", func(mux *router.ServeMux) {
			mux.Group("/", func(mux *router.ServeMux) {
				mux.Group("/", func(mux *router.ServeMux) {
					mux.Get("/", emptyHandler)
				})
			})

			currentPrefix := mux.CurrentPrefix()
			currentPattern := mux.CurrentPattern()

			mux.Group("/current", func(mux *router.ServeMux) {
				mux.Get("/prefix", func(w http.ResponseWriter, r *http.Request) {
					w.Write([]byte(currentPrefix))
				})

				mux.Get("/path", func(w http.ResponseWriter, r *http.Request) {
					w.Write([]byte(currentPattern))
				})
			})
		})
	})

	mux.Group("/handle", func(mux *router.ServeMux) {
		mux.Group("/all", func(mux *router.ServeMux) {
			mux.Any("/", emptyHandler)
		})
	})

	mux.Group("/same", func(mux *router.ServeMux) {
		mux.Group("/prefix", func(mux *router.ServeMux) {
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

	mux.Group("/name", func(mux *router.ServeMux) {
		mux.Name("named")

		mux.Get("/", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("named"))
		})
	})

	mux.Group("/before", func(mux *router.ServeMux) {
		mux.Group("/hook", func(mux *router.ServeMux) {
			mux.Group("/exact", func(mux *router.ServeMux) {
				mux.Before(func(next http.HandlerFunc) http.HandlerFunc {
					return func(w http.ResponseWriter, r *http.Request) {
						w.Write([]byte("abc"))

						return
					}
				})

				mux.Get("/", func(w http.ResponseWriter, r *http.Request) {
					w.Write([]byte("unreachable"))
				})
			})

			mux.Group("/prefix", func(mux *router.ServeMux) {
				mux.Get("/conflict", func(w http.ResponseWriter, r *http.Request) {
					w.Write([]byte("before conflict"))
				})

				mux.Group("/{foo}", func(mux *router.ServeMux) {
					mux.Before(func(next http.HandlerFunc) http.HandlerFunc {
						return func(w http.ResponseWriter, r *http.Request) {
							foo, _ := router.URLParam(r, "foo")
							if foo == "abc" {
								w.Write([]byte("123"))

								return
							}

							next(w, r)
						}
					})

					mux.Group("/bar", func(mux *router.ServeMux) {
						mux.Before(func(next http.HandlerFunc) http.HandlerFunc {
							return func(w http.ResponseWriter, r *http.Request) {
								foo, _ := router.URLParam(r, "foo")
								if foo == "qux" {
									w.Write([]byte("quxxxxx"))

									return
								}

								next(w, r)
							}
						})

						mux.Get("/", func(w http.ResponseWriter, r *http.Request) {
							param, _ := router.URLParam(r, "foo")

							w.Write([]byte(param))
						})
					})
				})
			})
		})
	})

	mux.Get("/url/{ ignore }/{ status }/qux", func(w http.ResponseWriter, r *http.Request) {
		status, _ := router.URLParam(r, "status")

		switch status {
		case "teapot":
			w.WriteHeader(http.StatusTeapot)

		case "gone":
			w.WriteHeader(http.StatusGone)

		default:
			w.WriteHeader(http.StatusBadRequest)
		}
	})

	mux.Group("/overlap-prefix", func(mux *router.ServeMux) {
		mux.Group("/{foo}", func(mux *router.ServeMux) {
			mux.Get("/", echoHandler)

			mux.Group("/overlap-suffix", func(mux *router.ServeMux) {
				mux.Get("/", func(w http.ResponseWriter, r *http.Request) {
					param, _ := router.URLParam(r, "foo")

					w.Write([]byte(param))
				})
			})
		})
	})

	mux.Get("/lazy/{ first}/rest/{rest }", func(w http.ResponseWriter, r *http.Request) {
		first, _ := router.URLParam(r, "first")
		rest, _ := router.URLParam(r, "rest")

		w.Write([]byte(first + "/" + rest))
	})

	mux.Get("/greedy/{first}/rest/{ rest ...}", func(w http.ResponseWriter, r *http.Request) {
		first, _ := router.URLParam(r, "first")
		rest, _ := router.URLParam(r, "rest")

		w.Write([]byte(first + "/" + rest))
	})

	mux.Get("/redirect/dst", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("redirected"))
	})

	mux.Group("/foo-prefix", func(mux *router.ServeMux) {
		// Redirects ignore the prefix
		mux.Redirect(http.MethodGet, "/redirect/src", "/redirect/dst", http.StatusTemporaryRedirect)
		mux.Redirect(http.MethodGet, "/{var}/redirect/src/var", "/{var}/dst", http.StatusTemporaryRedirect)
		mux.Redirect(http.MethodGet, "/{var}/{varfoo}/redirect/src/var", "/{var}/dst", http.StatusTemporaryRedirect)
	})

	mux.Get("/rewrite/dst", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("rewritten"))
	})

	mux.Group("/bar-prefix", func(mux *router.ServeMux) {
		// Rewrites ignore the prefix
		mux.Rewrite(http.MethodGet, "/rewrite/src", "/rewrite/dst")
		mux.Rewrite(http.MethodGet, "/{var}/rewrite/src/var", "/{var}/dst")
		mux.Rewrite(http.MethodGet, "/{var}/{varfoo}/rewrite/src/var", "/{var}/dst")
	})

	mux.Get("/aa/bb/cc/dd", echoHandler, "simple")
	mux.Get("/aa/{bb}/cc/{dd}", echoHandler, "complex")

	route := mux.Get("/a/{b}/c/{d}", func(w http.ResponseWriter, r *http.Request) {
		b, _ := router.URLParam(r, "b")
		d, _ := router.URLParam(r, "d")

		w.Write([]byte("/a/" + b + "/c/" + d))
	}, "foo.bar")

	mux.Post("/a/{b}/c/{d}", func(w http.ResponseWriter, r *http.Request) {
		b, _ := router.URLParam(r, "b")
		d, _ := router.URLParam(r, "d")

		w.Write([]byte("/a/" + b + "/c/" + d))
	}, "foo.bar.post")

	tt := []struct {
		name       string
		method     string
		path       string
		wantBody   string
		wantStatus int
	}{
		{"options method ok", http.MethodOptions, "/", "/", http.StatusOK},
		{"connect method ok", http.MethodConnect, "/", "/", http.StatusOK},
		{"trace method ok", http.MethodTrace, "/", "/", http.StatusOK},
		{"head method ok", http.MethodHead, "/", "", http.StatusOK},
		{"get method ok", http.MethodGet, "/", "/", http.StatusOK},
		{"post method ok", http.MethodPost, "/", "/", http.StatusOK},
		{"put method ok", http.MethodPut, "/", "/", http.StatusOK},
		{"patch method ok", http.MethodPatch, "/", "/", http.StatusOK},
		{"delete method ok", http.MethodDelete, "/", "/", http.StatusOK},

		{"options method ok", http.MethodOptions, "/route/test", "", http.StatusOK},
		{"connect method ok", http.MethodConnect, "/route/test", "", http.StatusOK},
		{"trace method ok", http.MethodTrace, "/route/test", "", http.StatusOK},
		{"head method ok", http.MethodHead, "/route/test", "", http.StatusOK},
		{"get method ok", http.MethodGet, "/route/test", "", http.StatusOK},
		{"post method ok", http.MethodPost, "/route/test", "", http.StatusOK},
		{"put method ok", http.MethodPut, "/route/test", "", http.StatusOK},
		{"patch method ok", http.MethodPatch, "/route/test", "", http.StatusOK},
		{"delete method ok", http.MethodDelete, "/route/test", "", http.StatusOK},
		{"get method with consecutive slashes ok", http.MethodGet, "/consecutive-slashes", "", http.StatusOK},

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
		{"dynamic url with lazy rest", http.MethodGet, "/lazy/foo/rest/baz", "foo/baz", http.StatusOK},
		{"dynamic url with empty lazy rest", http.MethodGet, "/lazy/foo/rest/", "foo/", http.StatusOK},
		{"dynamic url with lazy rest no match", http.MethodGet, "/lazy/foo/rest/baz/qux", "", http.StatusNotFound},
		{"dynamic url with greedy rest", http.MethodGet, "/greedy/foo/rest/baz/qux", "foo/baz/qux", http.StatusOK},
		{"dynamic url with empty greedy rest", http.MethodGet, "/greedy/foo/rest/", "foo/", http.StatusOK},
		{"dynamic url with greedy rest no match", http.MethodGet, "/greedy/foo/bar/rest/baz/qux", "", http.StatusNotFound},
		{"dynamic url overlap", http.MethodGet, "/overlap-prefix/bar/overlap-suffix", "bar", http.StatusOK},

		{"order independence no rest 1", http.MethodGet, "/order/foo", "lazy param: /order/foo", http.StatusOK},
		{"order independence no rest 2", http.MethodGet, "/order/static", "static: /order/static", http.StatusOK},
		{"order independence no rest 3", http.MethodGet, "/order/independent", "/order/independent", http.StatusOK},
		{"order independence no rest 4", http.MethodGet, "/order/independent/foo", "/order/independent/foo", http.StatusOK},
		{"order independence no rest 5", http.MethodGet, "/order/independent/foo/bar", "/order/independent/foo/bar", http.StatusOK},
		{"order independence rest overlap 1", http.MethodGet, "/order/foo/bar/baz/qux", "/order/foo/bar/baz/qux", http.StatusOK},
		{"order independence rest overlap 2", http.MethodGet, "/order/independent/foo/baz", "greedy rest: /order/independent/foo/baz", http.StatusOK},
		{"order independence rest overlap 3", http.MethodGet, "/order/independent/foo/bar/baz", "greedy rest: /order/independent/foo/bar/baz", http.StatusOK},
		{"order independence rest no overlap", http.MethodGet, "/order/independent/qux/quxx/quxxx", "greedy rest: /order/independent/qux/quxx/quxxx", http.StatusOK},
		{"order independence rest long overlap", http.MethodGet, "/order/independent/long/quxx/quxxx", "long greedy rest: /order/independent/long/quxx/quxxx", http.StatusOK},

		{"route string param replacement", http.MethodGet, route.Replace("{b}", "123", "{d}", "456"), "/a/123/c/456", http.StatusOK},
		{"mux object route string param replacement", http.MethodGet, mux.Route("foo.bar").Replace("{b}", "x", "{d}", "y"), "/a/x/c/y", http.StatusOK},
		{"mux object route string param replacement post method", http.MethodPost, mux.Route("foo.bar.post").Replace("{b}", "x", "{d}", "y"), "/a/x/c/y", http.StatusOK},

		{"mux object path simple", http.MethodGet, mux.Path("simple"), "/aa/bb/cc/dd", http.StatusOK},
		{"mux object path complex", http.MethodGet, mux.Path("complex", "{bb}", "xx", "{dd}", "yy"), "/aa/xx/cc/yy", http.StatusOK},
		{"mux object path named prefix", http.MethodGet, mux.Path("named"), "named", http.StatusOK},
		{"mux object current prefix", http.MethodGet, "/get/only/current/prefix", "/get/only/", http.StatusOK},
		{"mux object current path", http.MethodGet, "/get/only/current/path", "/get/only", http.StatusOK},

		{"exact before hook", http.MethodGet, "/before/hook/exact", "abc", http.StatusOK},

		{"prefix before hook no stop", http.MethodGet, "/before/hook/prefix/bar/bar", "bar", http.StatusOK},
		{"prefix before hook stop abc", http.MethodGet, "/before/hook/prefix/abc/bar", "123", http.StatusOK},
		{"prefix before hook stop qux", http.MethodGet, "/before/hook/prefix/qux/bar", "quxxxxx", http.StatusOK},
		{"prefix before hook conflict", http.MethodGet, "/before/hook/prefix/conflict", "before conflict", http.StatusOK},

		{"redirect get method ok", http.MethodGet, "/redirect/src", "redirected", http.StatusOK},
		{"redirect with dynamic param", http.MethodGet, "/redirect/redirect/src/var", "redirected", http.StatusOK},
		{"redirect with multiple dynamic params", http.MethodGet, "/redirect/foo/redirect/src/var", "redirected", http.StatusOK},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			req := errsx.Must(http.NewRequest(tc.method, ts.URL+tc.path, nil))
			res := errsx.Must(ts.Client().Do(req))

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

	ts.Client().CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}

	tt = []struct {
		name       string
		method     string
		path       string
		wantBody   string
		wantStatus int
	}{
		{"rewrite get method ok", http.MethodGet, "/rewrite/src", "rewritten", http.StatusOK},
		{"rewrite with dynamic param", http.MethodGet, "/rewrite/rewrite/src/var", "rewritten", http.StatusOK},
		{"rewrite with multiple dynamic params", http.MethodGet, "/rewrite/foo/rewrite/src/var", "rewritten", http.StatusOK},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			req := errsx.Must(http.NewRequest(tc.method, ts.URL+tc.path, nil))
			res := errsx.Must(ts.Client().Do(req))

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

	t.Run("panic on duplicate route patterns", func(t *testing.T) {
		defer func() {
			if recover() == nil {
				t.Error("want panic; got <nil>")
			}
		}()

		mux := router.NewServeMux()

		mux.Get("/one/two/three/four", emptyHandler)
		mux.Get("/one/two/three/four", emptyHandler)
	})

	t.Run("no panic on duplicate route patterns with different methods", func(t *testing.T) {
		defer func() {
			if recover() != nil {
				t.Error("want <nil>; got panic")
			}
		}()

		mux := router.NewServeMux()

		mux.Get("/one/two/three/four", emptyHandler)
		mux.Post("/one/two/three/four", emptyHandler)
	})

	t.Run("panic on duplicate route patterns with parameters", func(t *testing.T) {
		defer func() {
			if recover() == nil {
				t.Error("want panic; got <nil>")
			}
		}()

		mux := router.NewServeMux()

		mux.Get("/one/two/{foo}/four", emptyHandler)
		mux.Get("/one/two/{bar}/four", emptyHandler)
		mux.Post("/one/two/{baz}/four", emptyHandler)
	})

	t.Run("panic on duplicate route patterns with greedy parameters", func(t *testing.T) {
		defer func() {
			if recover() == nil {
				t.Error("want panic; got <nil>")
			}
		}()

		mux := router.NewServeMux()

		mux.Get("/one/two/{foo}/four/{bar...}", emptyHandler)
		mux.Post("/one/two/{foo}/four/{baz...}", emptyHandler)
	})

	t.Run("panic on duplicate route names", func(t *testing.T) {
		defer func() {
			if recover() == nil {
				t.Error("want panic; got <nil>")
			}
		}()

		mux := router.NewServeMux()

		mux.Get("/hello", emptyHandler, "hello")
		mux.Post("/hello", emptyHandler, "hello")
	})

	t.Run("panic on greedy param that is not the last one", func(t *testing.T) {
		defer func() {
			if recover() == nil {
				t.Error("want panic; got <nil>")
			}
		}()

		mux := router.NewServeMux()

		mux.Get("/hello/{foo...}/world", emptyHandler)
	})

	t.Run("panic on invalid route path parameter replacements", func(t *testing.T) {
		mux := router.NewServeMux()
		route := mux.Get("/{w}/x/y/{z}", emptyHandler)

		tt := []struct {
			name string
			list []any
		}{
			{"wrong number of elements", []any{"{w}"}},
			{"wrong order", []any{"1", "{w}"}},
			{"missing parameter", []any{"{w}", "1"}},
			{"unknown parameter", []any{"{w}", "1", "{x}", "2", "{z}", "3"}},
			{"empty argument", []any{"{w}", "1", "{z}", ""}},
		}
		for _, tc := range tt {
			t.Run(tc.name, func(t *testing.T) {
				defer func() {
					if recover() == nil {
						t.Error("want panic; got <nil>")
					}
				}()

				route.Replace(tc.list...)
			})
		}
	})

	t.Run("panic on non-existent path", func(t *testing.T) {
		defer func() {
			if recover() == nil {
				t.Error("want panic; got <nil>")
			}
		}()

		mux := router.NewServeMux()

		mux.Get("/hello", emptyHandler, "simple")

		mux.Path("foo")
	})

	t.Run("panic on invalid path calls", func(t *testing.T) {
		mux := router.NewServeMux()

		mux.Get("/{w}/x/y/{z}", emptyHandler, "complex")

		tt := []struct {
			name string
			list []any
		}{
			{"wrong number of elements", []any{"{w}"}},
			{"wrong order", []any{"1", "{w}"}},
			{"missing parameter", []any{"{w}", "1"}},
			{"unknown parameter", []any{"{w}", "1", "{x}", "2", "{z}", "3"}},
			{"empty argument", []any{"{w}", "1", "{z}", ""}},
			{"no arguments", nil},
		}
		for _, tc := range tt {
			t.Run(tc.name, func(t *testing.T) {
				defer func() {
					if recover() == nil {
						t.Error("want panic; got <nil>")
					}
				}()

				mux.Path("complex", tc.list...)
			})
		}
	})
}
