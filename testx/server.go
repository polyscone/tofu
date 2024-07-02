package testx

import (
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"testing"
)

type Server struct {
	*httptest.Server
}

func NewServer(t *testing.T, handler http.Handler) *Server {
	t.Helper()

	ts := httptest.NewServer(handler)

	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatal(err)
	}

	ts.Client().Jar = jar

	return &Server{Server: ts}
}

func (s *Server) FindCookie(t *testing.T, cookiesURL, name string) *http.Cookie {
	t.Helper()

	u, err := url.Parse(cookiesURL)
	if err != nil {
		t.Fatal(err)
	}

	for _, cookie := range s.Client().Jar.Cookies(u) {
		if cookie.Name == name {
			return cookie
		}
	}

	return nil
}
