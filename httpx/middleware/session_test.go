package middleware_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/polyscone/tofu/httpx/middleware"
	"github.com/polyscone/tofu/httpx/router"
	"github.com/polyscone/tofu/session"
	"github.com/polyscone/tofu/testutil"
)

func TestSession(t *testing.T) {
	sm := session.NewManager(session.NewJSONMemoryRepo(false))

	mux := router.NewServeMux()

	mux.Use(middleware.Session(sm, nil))

	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		for _, set := range r.URL.Query()["set"] {
			key, value, _ := strings.Cut(set, ":")

			sm.Set(ctx, key, value)
		}

		for _, get := range r.URL.Query()["get"] {
			key, want, _ := strings.Cut(get, ":")

			if want, got := want, sm.GetString(ctx, key); want != got {
				t.Errorf("want %q; got %q", want, got)
			}
		}

		for _, pop := range r.URL.Query()["pop"] {
			key, want, _ := strings.Cut(pop, ":")

			if want, got := want, sm.PopString(ctx, key); want != got {
				t.Errorf("want %q; got %q", want, got)
			}
		}

		if r.URL.Query().Has("destroy") {
			sm.Destroy(ctx)
		}
	})

	ts := testutil.NewServer(t, mux)
	defer ts.Close()

	var sessionCookie *http.Cookie

	t.Run("initial request without setting session values", func(t *testing.T) {
		res, err := ts.Client().Get(ts.URL)
		if err != nil {
			t.Errorf("want <nil>; got %q", err)
		}

		defer res.Body.Close()

		sessionCookie = ts.FindCookie(t, ts.URL, middleware.SessionCookieNameInsecure)

		if got, cmp := sessionCookie.Value, ""; got == cmp {
			t.Errorf("want different strings; got equal (%q)", got)
		}
	})

	t.Run("second request expects same seesion id", func(t *testing.T) {
		res, err := ts.Client().Get(ts.URL)
		if err != nil {
			t.Errorf("want <nil>; got %q", err)
		}

		defer res.Body.Close()

		c := ts.FindCookie(t, ts.URL, middleware.SessionCookieNameInsecure)

		if want, got := sessionCookie.Value, c.Value; want != got {
			t.Errorf("want %q; got %q", want, got)
		}
	})

	t.Run("test session values", func(t *testing.T) {
		res, err := ts.Client().Get(ts.URL + "?set=foo:bar&set=baz:qux")
		if err != nil {
			t.Errorf("want <nil>; got %q", err)
		}

		defer res.Body.Close()

		res, err = ts.Client().Get(ts.URL + "?get=foo:bar&set=baz:qux")
		if err != nil {
			t.Errorf("want <nil>; got %q", err)
		}

		defer res.Body.Close()

		res, err = ts.Client().Get(ts.URL + "?pop=foo:bar&set=baz:qux")
		if err != nil {
			t.Errorf("want <nil>; got %q", err)
		}

		defer res.Body.Close()

		res, err = ts.Client().Get(ts.URL + "?get=foo:&set=baz:")
		if err != nil {
			t.Errorf("want <nil>; got %q", err)
		}

		defer res.Body.Close()

		res, err = ts.Client().Get(ts.URL + "?pop=foo:&set=baz:")
		if err != nil {
			t.Errorf("want <nil>; got %q", err)
		}

		defer res.Body.Close()
	})

	t.Run("destroy multiple sessions and recreate a new session", func(t *testing.T) {
		res, err := ts.Client().Get(ts.URL + "?destroy")
		if err != nil {
			t.Errorf("want <nil>; got %q", err)
		}

		defer res.Body.Close()

		if got := ts.FindCookie(t, ts.URL, middleware.SessionCookieNameInsecure); got != nil {
			t.Errorf("want <nil>; got %v", got)
		}

		res, err = ts.Client().Get(ts.URL + "?destroy")
		if err != nil {
			t.Errorf("want <nil>; got %q", err)
		}

		defer res.Body.Close()

		if got := ts.FindCookie(t, ts.URL, middleware.SessionCookieNameInsecure); got != nil {
			t.Errorf("want <nil>; got %v", got)
		}

		res, err = ts.Client().Get(ts.URL)
		if err != nil {
			t.Errorf("want <nil>; got %q", err)
		}

		defer res.Body.Close()

		sessionCookie = ts.FindCookie(t, ts.URL, middleware.SessionCookieNameInsecure)

		if got, cmp := sessionCookie.Value, ""; got == cmp {
			t.Errorf("want different strings; got equal (%q)", got)
		}
	})
}
