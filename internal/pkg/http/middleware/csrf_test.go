package middleware_test

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/polyscone/tofu/internal/pkg/csrf"
	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/http/middleware"
	"github.com/polyscone/tofu/internal/pkg/http/router"
	"github.com/polyscone/tofu/internal/pkg/testutil"
)

func TestCSRF(t *testing.T) {
	mux := router.NewServeMux()

	mux.Use(middleware.CSRF(&middleware.CSRFConfig{Insecure: true}))

	mux.Connect("/", func(w http.ResponseWriter, r *http.Request) {})
	mux.Head("/", func(w http.ResponseWriter, r *http.Request) {})
	mux.Options("/", func(w http.ResponseWriter, r *http.Request) {})
	mux.Trace("/", func(w http.ResponseWriter, r *http.Request) {})
	mux.Get("/", func(w http.ResponseWriter, r *http.Request) {})
	mux.Post("/", func(w http.ResponseWriter, r *http.Request) {})
	mux.Put("/", func(w http.ResponseWriter, r *http.Request) {})
	mux.Patch("/", func(w http.ResponseWriter, r *http.Request) {})
	mux.Delete("/", func(w http.ResponseWriter, r *http.Request) {})

	mux.Post("/renew", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		csrf.RenewToken(ctx)
	})

	t.Run("sending no token", func(t *testing.T) {
		tt := []struct {
			name   string
			method string
			want   int
		}{
			{"connect requires a valid token", http.MethodConnect, http.StatusBadRequest},
			{"head does not require a token", http.MethodHead, http.StatusOK},
			{"options does not require a token", http.MethodOptions, http.StatusOK},
			{"trace does not require a token", http.MethodTrace, http.StatusOK},
			{"get does not require a token", http.MethodGet, http.StatusOK},
			{"post requires a valid token", http.MethodPost, http.StatusBadRequest},
			{"put requires a valid token", http.MethodPut, http.StatusBadRequest},
			{"patch requires a valid token", http.MethodPatch, http.StatusBadRequest},
			{"delete requires a valid token", http.MethodDelete, http.StatusBadRequest},
		}
		for _, tc := range tt {
			t.Run(tc.name, func(t *testing.T) {
				ts := testutil.NewServer(t, mux)
				defer ts.Close()

				req := errors.Must(http.NewRequest(tc.method, ts.URL+"/", nil))
				res := errors.Must(ts.Client().Do(req))

				defer res.Body.Close()

				if want, got := tc.want, res.StatusCode; want != got {
					t.Errorf("want %v; got %v", want, got)
				}
			})
		}
	})

	t.Run("send token in header", func(t *testing.T) {
		tt := []struct {
			name   string
			method string
			want   int
		}{
			{"connect with a valid token", http.MethodConnect, http.StatusOK},
			{"post with a valid token", http.MethodPost, http.StatusOK},
			{"put with a valid token", http.MethodPut, http.StatusOK},
			{"patch with a valid token", http.MethodPatch, http.StatusOK},
			{"delete with a valid token", http.MethodDelete, http.StatusOK},
		}
		for _, tc := range tt {
			t.Run(tc.name, func(t *testing.T) {
				ts := testutil.NewServer(t, mux)
				defer ts.Close()

				// Make sure a token cookie is set
				{
					req := errors.Must(http.NewRequest(http.MethodGet, ts.URL+"/", nil))
					res := errors.Must(ts.Client().Do(req))

					defer res.Body.Close()

					if want, got := http.StatusOK, res.StatusCode; want != got {
						t.Errorf("want %v; got %v", want, got)
					}
					if got, cmp := res.Header.Get("set-cookie"), ""; got == cmp {
						t.Errorf("want different strings; got equal (%q)", got)
					}
				}

				csrfCookie := ts.FindCookie(t, ts.URL+"/", middleware.CSRFTokenCookieName)

				req := errors.Must(http.NewRequest(tc.method, ts.URL+"/", nil))

				req.Header.Set("x-csrf-token", csrfCookie.Value)

				res := errors.Must(ts.Client().Do(req))

				defer res.Body.Close()

				if want, got := tc.want, res.StatusCode; want != got {
					t.Errorf("want %v; got %v", want, got)
				}
				if want, got := "", res.Header.Get("set-cookie"); want != got {
					t.Errorf("want %q; got %q", want, got)
				}
			})
		}
	})

	t.Run("send token in form values", func(t *testing.T) {
		tt := []struct {
			name   string
			method string
			want   int
		}{
			{"post with a valid token", http.MethodPost, http.StatusOK},
			{"put with a valid token", http.MethodPut, http.StatusOK},
			{"patch with a valid token", http.MethodPatch, http.StatusOK},
		}
		for _, tc := range tt {
			t.Run(tc.name, func(t *testing.T) {
				ts := testutil.NewServer(t, mux)
				defer ts.Close()

				// Make sure a token cookie is set
				{
					req := errors.Must(http.NewRequest(http.MethodGet, ts.URL+"/", nil))
					res := errors.Must(ts.Client().Do(req))

					defer res.Body.Close()

					if want, got := http.StatusOK, res.StatusCode; want != got {
						t.Errorf("want %v; got %v", want, got)
					}
					if got, cmp := res.Header.Get("set-cookie"), ""; got == cmp {
						t.Errorf("want different strings; got equal (%q)", got)
					}
				}

				csrfCookie := ts.FindCookie(t, ts.URL+"/", middleware.CSRFTokenCookieName)

				form := strings.NewReader(url.Values{"_csrf": {csrfCookie.Value}}.Encode())
				req := errors.Must(http.NewRequest(tc.method, ts.URL+"/", form))

				req.Header.Set("content-type", "application/x-www-form-urlencoded")

				res := errors.Must(ts.Client().Do(req))

				defer res.Body.Close()

				if want, got := tc.want, res.StatusCode; want != got {
					t.Errorf("want %v; got %v", want, got)
				}
				if want, got := "", res.Header.Get("set-cookie"); want != got {
					t.Errorf("want %q; got %q", want, got)
				}
			})
		}
	})

	t.Run("renew token in handler", func(t *testing.T) {
		ts := testutil.NewServer(t, mux)
		defer ts.Close()

		// Make sure a token cookie is set
		{
			req := errors.Must(http.NewRequest(http.MethodGet, ts.URL+"/", nil))
			res := errors.Must(ts.Client().Do(req))

			defer res.Body.Close()

			if want, got := http.StatusOK, res.StatusCode; want != got {
				t.Errorf("want %v; got %v", want, got)
			}
			if got, cmp := res.Header.Get("set-cookie"), ""; got == cmp {
				t.Errorf("want different strings; got equal (%q)", got)
			}
		}

		csrfCookie := ts.FindCookie(t, ts.URL+"/", middleware.CSRFTokenCookieName)

		req := errors.Must(http.NewRequest(http.MethodPost, ts.URL+"/renew", nil))

		req.Header.Set("x-csrf-token", csrfCookie.Value)

		res := errors.Must(ts.Client().Do(req))

		defer res.Body.Close()

		if want, got := http.StatusOK, res.StatusCode; want != got {
			t.Errorf("want %v; got %v", want, got)
		}
		if got, cmp := res.Header.Get("set-cookie"), ""; got == cmp {
			t.Errorf("want different strings; got equal (%q)", got)
		}

		csrfRenewedCookie := ts.FindCookie(t, ts.URL+"/", middleware.CSRFTokenCookieName)

		if got, cmp := csrfRenewedCookie.Value, csrfCookie.Value; got == cmp {
			t.Errorf("want different strings; got equal (%q)", got)
		}
	})
}
