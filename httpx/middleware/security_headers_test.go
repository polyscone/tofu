package middleware_test

import (
	"net/http"
	"testing"

	"github.com/polyscone/tofu/errsx"
	"github.com/polyscone/tofu/httpx/middleware"
	"github.com/polyscone/tofu/httpx/router"
	"github.com/polyscone/tofu/testx"
)

func TestSecurityHeaders(t *testing.T) {
	mux := router.NewServeMux()

	mux.Use(middleware.SecurityHeaders(nil))

	ts := testx.NewServer(t, mux)
	defer ts.Close()

	req := errsx.Must(http.NewRequest(http.MethodGet, ts.URL, nil))
	res := errsx.Must(ts.Client().Do(req))

	defer res.Body.Close()

	if want, got := "nosniff", res.Header.Get("x-content-type-options"); want != got {
		t.Errorf("want %q; got %q", want, got)
	}
	if want, got := "deny", res.Header.Get("x-frame-options"); want != got {
		t.Errorf("want %q; got %q", want, got)
	}
}
