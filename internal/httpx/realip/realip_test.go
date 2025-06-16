package realip_test

import (
	"errors"
	"net/http"
	"testing"

	"github.com/polyscone/tofu/internal/errsx"
	"github.com/polyscone/tofu/internal/httpx/realip"
)

func TestFromRequest(t *testing.T) {
	t.Run("no headers set", func(t *testing.T) {
		req := errsx.Must(http.NewRequest(http.MethodGet, "/", nil))

		req.RemoteAddr = "1.2.3.4"

		if want, got := "1.2.3.4", errsx.Must(realip.FromRequest(req, nil)); want != got {
			t.Errorf("want %q; got %q", want, got)
		}
	})

	t.Run("x-forwarded-for header set with no trusted proxies", func(t *testing.T) {
		req := errsx.Must(http.NewRequest(http.MethodGet, "/", nil))

		req.RemoteAddr = "1.2.3.4"

		req.Header.Add("x-forwarded-for", "1.1.1.1, 2.2.2.2")
		req.Header.Add("x-forwarded-for", "3.3.3.3")
		req.Header.Add("x-forwarded-for", "4.4.4.4, 5.5.5.5, 6.6.6.6")

		if want, got := "1.2.3.4", errsx.Must(realip.FromRequest(req, nil)); want != got {
			t.Errorf("want %q; got %q", want, got)
		}
	})

	t.Run("x-forwarded-for header set with unmatched trusted proxies (remote addr)", func(t *testing.T) {
		req := errsx.Must(http.NewRequest(http.MethodGet, "/", nil))

		req.RemoteAddr = "1.2.3.4"

		req.Header.Add("x-forwarded-for", "1.1.1.1, 2.2.2.2")
		req.Header.Add("x-forwarded-for", "3.3.3.3")
		req.Header.Add("x-forwarded-for", "4.4.4.4, 5.5.5.5, 6.6.6.6")

		proxies := []string{"6.6.6.6", "2.2.2.2", "4.4.4.4", "5.5.5.5"}

		if want, got := "1.2.3.4", errsx.Must(realip.FromRequest(req, proxies)); want != got {
			t.Errorf("want %q; got %q", want, got)
		}
	})

	t.Run("x-forwarded-for header set with matched trusted proxies", func(t *testing.T) {
		req := errsx.Must(http.NewRequest(http.MethodGet, "/", nil))

		req.RemoteAddr = "1.2.3.4"

		req.Header.Add("x-forwarded-for", "1.1.1.1, 2.2.2.2")
		req.Header.Add("x-forwarded-for", "3.3.3.3")
		req.Header.Add("x-forwarded-for", "4.4.4.4, 5.5.5.5, 6.6.6.6")

		proxies := []string{"6.6.6.6", "2.2.2.2", "4.4.4.4", "5.5.5.5", "1.2.3.4"}

		if want, got := "3.3.3.3", errsx.Must(realip.FromRequest(req, proxies)); want != got {
			t.Errorf("want %q; got %q", want, got)
		}
	})

	t.Run("x-forwarded-for header set with all matched trusted proxies", func(t *testing.T) {
		req := errsx.Must(http.NewRequest(http.MethodGet, "/", nil))

		req.RemoteAddr = "1.2.3.4"

		req.Header.Add("x-forwarded-for", "1.1.1.1, 2.2.2.2")
		req.Header.Add("x-forwarded-for", "3.3.3.3")
		req.Header.Add("x-forwarded-for", "4.4.4.4, 5.5.5.5, 6.6.6.6")

		proxies := []string{"1.1.1.1", "3.3.3.3", "6.6.6.6", "2.2.2.2", "4.4.4.4", "5.5.5.5", "1.2.3.4"}

		if want, got := "1.1.1.1", errsx.Must(realip.FromRequest(req, proxies)); want != got {
			t.Errorf("want %q; got %q", want, got)
		}
	})

	t.Run("remote addr has a port", func(t *testing.T) {
		req := errsx.Must(http.NewRequest(http.MethodGet, "/", nil))

		req.RemoteAddr = "1.2.3.4:8080"

		if want, got := "1.2.3.4", errsx.Must(realip.FromRequest(req, nil)); want != got {
			t.Errorf("want %q; got %q", want, got)
		}
	})

	t.Run("remote addr has a port with trusted proxied and x-forwarded-for", func(t *testing.T) {
		req := errsx.Must(http.NewRequest(http.MethodGet, "/", nil))

		req.RemoteAddr = "1.2.3.4:8080"

		req.Header.Add("x-forwarded-for", "1.1.1.1, 2.2.2.2")

		proxies := []string{"1.2.3.4", "2.2.2.2"}

		if want, got := "1.1.1.1", errsx.Must(realip.FromRequest(req, proxies)); want != got {
			t.Errorf("want %q; got %q", want, got)
		}
	})

	t.Run("too many addresses", func(t *testing.T) {
		req := errsx.Must(http.NewRequest(http.MethodGet, "/", nil))

		req.RemoteAddr = "1.2.3.4"

		for range 50 {
			req.Header.Add("x-forwarded-for", "0.0.0.0")
		}

		_, err := realip.FromRequest(req, []string{"1.1.1.1"})
		if want, got := realip.ErrTooManyAddresses, err; !errors.Is(got, want) {
			t.Errorf("want realip.ErrTooManyAddresses; got %q", got)
		}
	})

	t.Run("last address is empty in x-forwarded-for", func(t *testing.T) {
		req := errsx.Must(http.NewRequest(http.MethodGet, "/", nil))

		req.RemoteAddr = "1.2.3.4"

		req.Header.Add("x-forwarded-for", ",,1.1.1.1,,2.2.2.2,,")

		proxies := []string{"1.2.3.4", "2.2.2.2"}

		if want, got := "1.1.1.1", errsx.Must(realip.FromRequest(req, proxies)); want != got {
			t.Errorf("want %q; got %q", want, got)
		}
	})

	t.Run("middle address is empty in x-forwarded-for", func(t *testing.T) {
		req := errsx.Must(http.NewRequest(http.MethodGet, "/", nil))

		req.RemoteAddr = "1.2.3.4"

		req.Header.Add("x-forwarded-for", "1.1.1.1,,2.2.2.2,,")

		proxies := []string{"1.2.3.4", "2.2.2.2"}

		if want, got := "1.1.1.1", errsx.Must(realip.FromRequest(req, proxies)); want != got {
			t.Errorf("want %q; got %q", want, got)
		}
	})

	t.Run("last IP address is invalid in x-forwarded-for", func(t *testing.T) {
		req := errsx.Must(http.NewRequest(http.MethodGet, "/", nil))

		req.RemoteAddr = "1.2.3.4"

		req.Header.Add("x-forwarded-for", "not-an-ip,foo, 2.2.2.2")

		proxies := []string{"1.2.3.4", "2.2.2.2"}

		if want, got := "2.2.2.2", errsx.Must(realip.FromRequest(req, proxies)); want != got {
			t.Errorf("want %q; got %q", want, got)
		}
	})

	t.Run("middle IP address is invalid in x-forwarded-for", func(t *testing.T) {
		req := errsx.Must(http.NewRequest(http.MethodGet, "/", nil))

		req.RemoteAddr = "1.2.3.4"

		req.Header.Add("x-forwarded-for", "1.1.1.1, not-an-ip, 2.2.2.2")

		proxies := []string{"1.2.3.4", "2.2.2.2"}

		if want, got := "1.1.1.1", errsx.Must(realip.FromRequest(req, proxies)); want != got {
			t.Errorf("want %q; got %q", want, got)
		}
	})

	t.Run("ports in x-forwarded-for", func(t *testing.T) {
		req := errsx.Must(http.NewRequest(http.MethodGet, "/", nil))

		req.RemoteAddr = "1.2.3.4:8080"

		req.Header.Add("x-forwarded-for", "1.1.1.1:9090, 2.2.2.2:3030")

		proxies := []string{"1.2.3.4", "2.2.2.2"}

		if want, got := "1.1.1.1", errsx.Must(realip.FromRequest(req, proxies)); want != got {
			t.Errorf("want %q; got %q", want, got)
		}
	})

	t.Run("bracketed IPv6 for trusted proxy", func(t *testing.T) {
		req := errsx.Must(http.NewRequest(http.MethodGet, "/", nil))

		req.RemoteAddr = "1.2.3.4"

		req.Header.Add("x-forwarded-for", "[::1]")

		proxies := []string{"1.2.3.4", "::1"}

		if want, got := "::1", errsx.Must(realip.FromRequest(req, proxies)); want != got {
			t.Errorf("want %q; got %q", want, got)
		}
	})
}
