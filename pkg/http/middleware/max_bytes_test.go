package middleware_test

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/polyscone/tofu/pkg/errsx"
	"github.com/polyscone/tofu/pkg/http/middleware"
	"github.com/polyscone/tofu/pkg/http/router"
	"github.com/polyscone/tofu/pkg/testutil"
)

func TestMaxBytes(t *testing.T) {
	mux := router.NewServeMux()

	mux.Use(middleware.MaxBytes(func(r *http.Request) int {
		switch r.Method {
		case http.MethodPost, http.MethodPut, http.MethodPatch:
			return 10
		}

		return 0
	}))

	readJSON := func(w http.ResponseWriter, r *http.Request) {
		var maxBytesError *http.MaxBytesError

		_, err := io.ReadAll(r.Body)
		if err != nil && errors.As(err, &maxBytesError) {
			w.WriteHeader(http.StatusRequestEntityTooLarge)
		}
	}

	mux.HandleFunc("GET /get", readJSON)
	mux.HandleFunc("POST /post", readJSON)
	mux.HandleFunc("PUT /put", readJSON)
	mux.HandleFunc("PATCH /patch", readJSON)
	mux.HandleFunc("DELETE /delete", readJSON)

	ts := testutil.NewServer(t, mux)
	defer ts.Close()

	tt := []struct {
		name   string
		method string
		path   string
		body   string
		want   int
	}{
		{"small request body on get", http.MethodGet, "/get", "", http.StatusOK},
		{"small request body on post", http.MethodPost, "/post", "Small", http.StatusOK},
		{"small request body on put", http.MethodPut, "/put", "Small", http.StatusOK},
		{"small request body on patch", http.MethodPatch, "/patch", "Small", http.StatusOK},
		{"small request body on delete", http.MethodDelete, "/delete", "", http.StatusOK},

		{"too large request body on get", http.MethodGet, "/get", "a", http.StatusRequestEntityTooLarge},
		{"too large request body on post", http.MethodPost, "/post", "Body that is too large", http.StatusRequestEntityTooLarge},
		{"too large request body on put", http.MethodPut, "/put", "Body that is too large", http.StatusRequestEntityTooLarge},
		{"too large request body on patch", http.MethodPatch, "/patch", "Body that is too large", http.StatusRequestEntityTooLarge},
		{"too large request body on delete", http.MethodDelete, "/delete", "a", http.StatusRequestEntityTooLarge},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			req := errsx.Must(http.NewRequest(tc.method, ts.URL+tc.path, strings.NewReader(tc.body)))
			res := errsx.Must(ts.Client().Do(req))

			defer res.Body.Close()

			if want, got := tc.want, res.StatusCode; want != got {
				t.Errorf("want %v; got %v", want, got)
			}
		})
	}
}
