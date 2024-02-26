package middleware_test

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/polyscone/tofu/internal/pkg/csrf"
	"github.com/polyscone/tofu/internal/pkg/errsx"
	"github.com/polyscone/tofu/internal/pkg/http/middleware"
	"github.com/polyscone/tofu/internal/pkg/http/router"
	"github.com/polyscone/tofu/internal/pkg/testutil"
)

func TestCSRF(t *testing.T) {
	mux := router.NewServeMux()

	mux.Use(middleware.CSRF(nil))

	mux.HandleFunc("CONNECT /", func(w http.ResponseWriter, r *http.Request) {})
	mux.HandleFunc("HEAD /", func(w http.ResponseWriter, r *http.Request) {})
	mux.HandleFunc("OPTIONS /", func(w http.ResponseWriter, r *http.Request) {})
	mux.HandleFunc("TRACE /", func(w http.ResponseWriter, r *http.Request) {})
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {})
	mux.HandleFunc("POST /", func(w http.ResponseWriter, r *http.Request) {})
	mux.HandleFunc("PUT /", func(w http.ResponseWriter, r *http.Request) {})
	mux.HandleFunc("PATCH /", func(w http.ResponseWriter, r *http.Request) {})
	mux.HandleFunc("DELETE /", func(w http.ResponseWriter, r *http.Request) {})

	mux.HandleFunc("POST /renew", func(w http.ResponseWriter, r *http.Request) {
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

				req := errsx.Must(http.NewRequest(tc.method, ts.URL+"/", nil))
				res := errsx.Must(ts.Client().Do(req))

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
					req := errsx.Must(http.NewRequest(http.MethodGet, ts.URL+"/", nil))
					res := errsx.Must(ts.Client().Do(req))

					defer res.Body.Close()

					if want, got := http.StatusOK, res.StatusCode; want != got {
						t.Errorf("want %v; got %v", want, got)
					}
					if got, cmp := res.Header.Get("set-cookie"), ""; got == cmp {
						t.Errorf("want different strings; got equal (%q)", got)
					}
				}

				csrfCookie := ts.FindCookie(t, ts.URL+"/", middleware.CSRFTokenCookieNameInsecure)

				req := errsx.Must(http.NewRequest(tc.method, ts.URL+"/", nil))

				req.Header.Set("x-csrf-token", csrfCookie.Value)

				res := errsx.Must(ts.Client().Do(req))

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
					req := errsx.Must(http.NewRequest(http.MethodGet, ts.URL+"/", nil))
					res := errsx.Must(ts.Client().Do(req))

					defer res.Body.Close()

					if want, got := http.StatusOK, res.StatusCode; want != got {
						t.Errorf("want %v; got %v", want, got)
					}
					if got, cmp := res.Header.Get("set-cookie"), ""; got == cmp {
						t.Errorf("want different strings; got equal (%q)", got)
					}
				}

				csrfCookie := ts.FindCookie(t, ts.URL+"/", middleware.CSRFTokenCookieNameInsecure)

				form := strings.NewReader(url.Values{"_csrf": {csrfCookie.Value}}.Encode())
				req := errsx.Must(http.NewRequest(tc.method, ts.URL+"/", form))

				req.Header.Set("content-type", "application/x-www-form-urlencoded")

				res := errsx.Must(ts.Client().Do(req))

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
			req := errsx.Must(http.NewRequest(http.MethodGet, ts.URL+"/", nil))
			res := errsx.Must(ts.Client().Do(req))

			defer res.Body.Close()

			if want, got := http.StatusOK, res.StatusCode; want != got {
				t.Errorf("want %v; got %v", want, got)
			}
			if got, cmp := res.Header.Get("set-cookie"), ""; got == cmp {
				t.Errorf("want different strings; got equal (%q)", got)
			}
		}

		csrfCookie := ts.FindCookie(t, ts.URL+"/", middleware.CSRFTokenCookieNameInsecure)

		req := errsx.Must(http.NewRequest(http.MethodPost, ts.URL+"/renew", nil))

		req.Header.Set("x-csrf-token", csrfCookie.Value)

		res := errsx.Must(ts.Client().Do(req))

		defer res.Body.Close()

		if want, got := http.StatusOK, res.StatusCode; want != got {
			t.Errorf("want %v; got %v", want, got)
		}
		if got, cmp := res.Header.Get("set-cookie"), ""; got == cmp {
			t.Errorf("want different strings; got equal (%q)", got)
		}

		csrfRenewedCookie := ts.FindCookie(t, ts.URL+"/", middleware.CSRFTokenCookieNameInsecure)

		if got, cmp := csrfRenewedCookie.Value, csrfCookie.Value; got == cmp {
			t.Errorf("want different strings; got equal (%q)", got)
		}
	})
}
