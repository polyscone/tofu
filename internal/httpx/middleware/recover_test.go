package middleware_test

import (
	"net/http"
	"testing"

	"github.com/polyscone/tofu/internal/errsx"
	"github.com/polyscone/tofu/internal/httpx/middleware"
	"github.com/polyscone/tofu/internal/httpx/router"
	"github.com/polyscone/tofu/internal/testx"
)

func TestRecover(t *testing.T) {
	mux := router.NewServeMux()

	mux.Use(middleware.Recover(nil))

	mux.HandleFunc("GET /ok", func(w http.ResponseWriter, r *http.Request) {})

	mux.HandleFunc("GET /panic", func(w http.ResponseWriter, r *http.Request) {
		panic("test panic here")
	})

	mux.HandleFunc("GET /nil-pointer", func(w http.ResponseWriter, r *http.Request) {
		type Foo struct{ Bar string }

		var foo *Foo

		foo.Bar = "123"
	})

	ts := testx.NewServer(t, mux)
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
		t.Run(tc.name, func(t *testing.T) {
			req := errsx.Must(http.NewRequest(tc.method, ts.URL+tc.path, nil))
			res := errsx.Must(ts.Client().Do(req))

			defer res.Body.Close()

			if want, got := tc.want, res.StatusCode; want != got {
				t.Errorf("want %v; got %v", want, got)
			}
		})
	}
}
