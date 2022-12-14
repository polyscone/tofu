package middleware_test

import (
	"net/http"
	"testing"

	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/http/middleware"
	"github.com/polyscone/tofu/internal/pkg/http/router"
	"github.com/polyscone/tofu/internal/pkg/testutil"
)

func TestSecurityHeaders(t *testing.T) {
	mux := router.NewServeMux()

	mux.Use(middleware.SecurityHeaders)

	ts := testutil.NewServer(t, mux)
	defer ts.Close()

	req := errors.Must(http.NewRequest(http.MethodGet, ts.URL, nil))
	res := errors.Must(ts.Client().Do(req))

	defer res.Body.Close()

	if want, got := "nosniff", res.Header.Get("x-content-type-options"); want != got {
		t.Errorf("want %q; got %q", want, got)
	}
	if want, got := "deny", res.Header.Get("x-frame-options"); want != got {
		t.Errorf("want %q; got %q", want, got)
	}
}
