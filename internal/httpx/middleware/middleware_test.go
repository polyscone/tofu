package middleware_test

import (
	"net/http"
	"testing"

	"github.com/polyscone/tofu/internal/httpx/middleware"
)

func TestApply(t *testing.T) {
	var values []int

	mw1 := func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			values = append(values, 0)

			next(w, r)
		}
	}

	mw2 := func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			values = append(values, 1)

			next(w, r)
		}
	}

	mw3 := func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			values = append(values, 2)

			next(w, r)
		}
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		values = append(values, 3)
	})

	stack := middleware.Apply(handler, mw1, mw2, mw3)

	stack.ServeHTTP(nil, nil)

	want, got := []int{0, 1, 2, 3}, values
	if len(want) != len(got) {
		t.Fatalf("want slice of length %v; got length %v", len(want), len(got))
	}
	for i, a := range want {
		if b := got[i]; a != b {
			t.Fatalf("\nwant %#v\ngot  %#v", want, got)
		}
	}
}
