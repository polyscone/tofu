package middleware_test

import (
	"net/http"
	"testing"

	"github.com/polyscone/tofu/errsx"
	"github.com/polyscone/tofu/httpx/middleware"
	"github.com/polyscone/tofu/httpx/router"
	"github.com/polyscone/tofu/testutil"
)

func TestNoContent(t *testing.T) {
	mux := router.NewServeMux()

	mux.Use(middleware.NoContent)

	mux.HandleFunc("GET /empty", func(w http.ResponseWriter, r *http.Request) {})

	mux.HandleFunc("GET /populated", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello, World!"))
	})

	ts := testutil.NewServer(t, mux)
	defer ts.Close()

	tt := []struct {
		name   string
		method string
		path   string
		want   int
	}{
		{"no content", http.MethodGet, "/empty", http.StatusNoContent},
		{"some content", http.MethodGet, "/populated", http.StatusOK},
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
