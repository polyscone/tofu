package middleware_test

import (
	"net/http"
	"testing"

	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/http/middleware"
	"github.com/polyscone/tofu/internal/pkg/http/router"
	"github.com/polyscone/tofu/internal/pkg/testutil"
)

func TestRecover(t *testing.T) {
	mux := router.NewServeMux()

	mux.Use(middleware.Recover(nil))

	mux.Get("/ok", func(w http.ResponseWriter, r *http.Request) {})

	mux.Get("/panic", func(w http.ResponseWriter, r *http.Request) {
		panic("test panic here")
	})

	mux.Get("/nil-pointer", func(w http.ResponseWriter, r *http.Request) {
		type Foo struct{ Bar string }

		var foo *Foo

		foo.Bar = "123"
	})

	ts := testutil.NewServer(t, mux)
	defer ts.Close()

	tt := []struct {
		name   string
		method string
		path   string
		want   int
	}{
		{"no error/panic", http.MethodGet, "/ok", http.StatusOK},
		{"explicit panic", http.MethodGet, "/panic", http.StatusInternalServerError},
		{"nil pointer dereference", http.MethodGet, "/nil-pointer", http.StatusInternalServerError},
	}
	for _, tc := range tt {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			req := errors.Must(http.NewRequest(tc.method, ts.URL+tc.path, nil))
			res := errors.Must(ts.Client().Do(req))

			defer res.Body.Close()

			if want, got := tc.want, res.StatusCode; want != got {
				t.Errorf("want %v; got %v", want, got)
			}
		})
	}
}
