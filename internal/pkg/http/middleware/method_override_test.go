package middleware_test

import (
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/http/middleware"
	"github.com/polyscone/tofu/internal/pkg/http/router"
	"github.com/polyscone/tofu/internal/pkg/testutil"
)

func TestMethodOverride(t *testing.T) {
	mux := router.NewServeMux()

	mux.Use(middleware.MethodOverride)

	mux.Post("/", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(http.MethodPost)) })
	mux.Put("/", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(http.MethodPut)) })
	mux.Patch("/", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(http.MethodPatch)) })
	mux.Delete("/", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(http.MethodDelete)) })

	ts := testutil.NewServer(t, mux)
	defer ts.Close()

	tt := []struct {
		name   string
		form   string
		header string
		want   string
	}{
		{"post override form value", http.MethodPost, "", http.MethodPost},
		{"put override form value", http.MethodPut, "", http.MethodPut},
		{"patch override form value", http.MethodPatch, "", http.MethodPatch},
		{"delete override form value", http.MethodDelete, "", http.MethodDelete},

		{"post override header value", "", http.MethodPost, http.MethodPost},
		{"put override header value", "", http.MethodPut, http.MethodPut},
		{"patch override header value", "", http.MethodPatch, http.MethodPatch},
		{"delete override header value", "", http.MethodDelete, http.MethodDelete},

		{"post override form value preferred over header value", http.MethodPost, http.MethodPut, http.MethodPost},
		{"put override form value preferred over header value", http.MethodPut, http.MethodPatch, http.MethodPut},
		{"patch override form value preferred over header value", http.MethodPatch, http.MethodDelete, http.MethodPatch},
		{"delete override form value preferred over header value", http.MethodDelete, http.MethodPost, http.MethodDelete},

		{"ignore options override", http.MethodOptions, http.MethodOptions, http.MethodPost},
		{"ignore connect override", http.MethodConnect, http.MethodConnect, http.MethodPost},
		{"ignore trace override", http.MethodTrace, http.MethodTrace, http.MethodPost},
		{"ignore head override", http.MethodHead, http.MethodHead, http.MethodPost},
		{"ignore get override", http.MethodGet, http.MethodGet, http.MethodPost},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			var form io.Reader

			if tc.form != "" {
				form = strings.NewReader(url.Values{"_method": {tc.form}}.Encode())
			}

			req := errors.Must(http.NewRequest(http.MethodPost, ts.URL+"/?want="+tc.want, form))

			req.Header.Set("content-type", "application/x-www-form-urlencoded")

			if tc.header != "" {
				req.Header.Set("x-http-method-override", tc.header)
			}

			res := errors.Must(ts.Client().Do(req))

			defer res.Body.Close()

			got := string(errors.Must(io.ReadAll(res.Body)))
			if want := tc.want; want != got {
				t.Errorf("want %v; got %v", want, got)
			}
		})
	}
}
