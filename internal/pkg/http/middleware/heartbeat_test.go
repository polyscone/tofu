package middleware_test

import (
	"net/http"
	"testing"

	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/http/middleware"
	"github.com/polyscone/tofu/internal/pkg/http/router"
	"github.com/polyscone/tofu/internal/pkg/testutil"
)

func TestHeartbeat(t *testing.T) {
	mux := router.NewServeMux()

	mux.Use(middleware.Heartbeat("/health"))

	ts := testutil.NewServer(t, mux)
	defer ts.Close()

	tt := []struct {
		name   string
		method string
		path   string
		want   int
	}{
		{"status 200 ok for get", http.MethodGet, "/health", http.StatusOK},
		{"status 204 no content for head", http.MethodHead, "/health", http.StatusNoContent},
		{"status 404 not found on fallthrough", http.MethodGet, "/foo", http.StatusNotFound},
	}
	for _, tc := range tt {
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
