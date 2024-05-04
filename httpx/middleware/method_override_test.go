package middleware_test

import (
	"io"
	"net/http"
	"testing"

	"github.com/polyscone/tofu/errsx"
	"github.com/polyscone/tofu/httpx/middleware"
	"github.com/polyscone/tofu/httpx/router"
	"github.com/polyscone/tofu/testutil"
)

func TestMethodOverride(t *testing.T) {
	mux := router.NewServeMux()

	mux.Use(middleware.MethodOverride)

	mux.HandleFunc("POST /", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(http.MethodPost)) })
	mux.HandleFunc("PUT /", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(http.MethodPut)) })
	mux.HandleFunc("PATCH /", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(http.MethodPatch)) })
	mux.HandleFunc("DELETE /", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(http.MethodDelete)) })

	ts := testutil.NewServer(t, mux)
	defer ts.Close()

	tt := []struct {
		name   string
		query  string
		header string
		want   string
	}{
		{"post override query value", http.MethodPost, "", http.MethodPost},
		{"put override query value", http.MethodPut, "", http.MethodPut},
		{"patch override query value", http.MethodPatch, "", http.MethodPatch},
		{"delete override query value", http.MethodDelete, "", http.MethodDelete},

		{"post override header value", "", http.MethodPost, http.MethodPost},
		{"put override header value", "", http.MethodPut, http.MethodPut},
		{"patch override header value", "", http.MethodPatch, http.MethodPatch},
		{"delete override header value", "", http.MethodDelete, http.MethodDelete},

		{"post override query value preferred over header value", http.MethodPost, http.MethodPut, http.MethodPost},
		{"put override query value preferred over header value", http.MethodPut, http.MethodPatch, http.MethodPut},
		{"patch override query value preferred over header value", http.MethodPatch, http.MethodDelete, http.MethodPatch},
		{"delete override query value preferred over header value", http.MethodDelete, http.MethodPost, http.MethodDelete},

		{"ignore options override", http.MethodOptions, http.MethodOptions, http.MethodPost},
		{"ignore connect override", http.MethodConnect, http.MethodConnect, http.MethodPost},
		{"ignore trace override", http.MethodTrace, http.MethodTrace, http.MethodPost},
		{"ignore head override", http.MethodHead, http.MethodHead, http.MethodPost},
		{"ignore get override", http.MethodGet, http.MethodGet, http.MethodPost},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			var query string
			if tc.query != "" {
				query = "&_method=" + tc.query
			}

			req := errsx.Must(http.NewRequest(http.MethodPost, ts.URL+"/?want="+tc.want+query, nil))

			req.Header.Set("content-type", "application/x-www-form-urlencoded")

			if tc.header != "" {
				req.Header.Set("x-http-method-override", tc.header)
			}

			res := errsx.Must(ts.Client().Do(req))

			defer res.Body.Close()

			got := string(errsx.Must(io.ReadAll(res.Body)))
			if want := tc.want; want != got {
				t.Errorf("want %v; got %v", want, got)
			}
		})
	}
}
