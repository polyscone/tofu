package handler

import (
	"net/http"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/polyscone/tofu/internal/adapter/web/passport"
	"github.com/polyscone/tofu/internal/adapter/web/sess"
)

type IsAuthorisedFunc func(passport passport.Passport) bool
type RedirectFunc func() string

type prefixGuard struct {
	path         string
	isAuthorised IsAuthorisedFunc
}

type Guard struct {
	svc      *Services
	exact    map[string]IsAuthorisedFunc
	prefixes []prefixGuard
	redirect RedirectFunc
}

func NewGuard(svc *Services, redirect RedirectFunc) *Guard {
	return &Guard{
		svc:      svc,
		redirect: redirect,
		exact:    make(map[string]IsAuthorisedFunc),
	}
}

func (g *Guard) isAuthenticated(passport passport.Passport) bool {
	return passport.IsAuthenticated()
}

func (g *Guard) ProtectFunc(path string, isAuthorised IsAuthorisedFunc) {
	g.exact[path] = isAuthorised
}

func (g *Guard) Protect(path string) {
	g.ProtectFunc(path, g.isAuthenticated)
}

func (g *Guard) ProtectPrefixFunc(path string, isAuthorised IsAuthorisedFunc) {
	g.prefixes = append(g.prefixes, prefixGuard{
		path:         path,
		isAuthorised: isAuthorised,
	})

	sort.Slice(g.prefixes, func(i, j int) bool {
		// Reverse string length sort so the longest key comes first
		return utf8.RuneCountInString(g.prefixes[j].path) < utf8.RuneCountInString(g.prefixes[i].path)
	})
}

func (g *Guard) ProtectPrefix(path string) {
	g.ProtectPrefixFunc(path, g.isAuthenticated)
}

func (g *Guard) Middleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Exact path matches should take precedence over more
		// generic prefix matches
		if isAuthorised, ok := g.exact[r.URL.Path]; ok {
			ctx := r.Context()

			passport := g.svc.Passport(ctx)
			if !isAuthorised(passport) {
				g.svc.Sessions.Set(ctx, sess.Redirect, r.URL.String())

				http.Redirect(w, r, g.redirect(), http.StatusSeeOther)

				return
			}
		}

		for _, guard := range g.prefixes {
			if !strings.HasPrefix(r.URL.Path, guard.path) {
				continue
			}

			ctx := r.Context()

			passport := g.svc.Passport(ctx)
			if !guard.isAuthorised(passport) {
				g.svc.Sessions.Set(ctx, sess.Redirect, r.URL.String())

				http.Redirect(w, r, g.redirect(), http.StatusSeeOther)

				return
			}
		}

		next(w, r)
	}
}
