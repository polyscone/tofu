package handler

import (
	"net/http"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/polyscone/tofu/internal/adapter/web/passport"
	"github.com/polyscone/tofu/internal/adapter/web/sess"
	"github.com/polyscone/tofu/internal/pkg/errors"
)

var ErrRedirect = errors.New("redirect")

type CheckAuthorisedFunc func(passport passport.Passport) error
type RedirectFunc func() string

type prefixGuard struct {
	path            string
	checkAuthorised CheckAuthorisedFunc
}

type Guard struct {
	svc      *Services
	exact    map[string]CheckAuthorisedFunc
	prefixes []prefixGuard
	redirect RedirectFunc
}

func NewGuard(svc *Services, redirect RedirectFunc) *Guard {
	return &Guard{
		svc:      svc,
		redirect: redirect,
		exact:    make(map[string]CheckAuthorisedFunc),
	}
}

func (g *Guard) isSignedIn(passport passport.Passport) error {
	if !passport.IsSignedIn() {
		return errors.Tracef(ErrRedirect)
	}

	return nil
}

func (g *Guard) Protect(path string, checkAuthorised CheckAuthorisedFunc) {
	g.exact[path] = checkAuthorised
}

func (g *Guard) ProtectPrefix(path string, checkAuthorised CheckAuthorisedFunc) {
	path = strings.TrimSuffix(path, "/")

	g.Protect(path, checkAuthorised)

	g.prefixes = append(g.prefixes, prefixGuard{
		path:            path + "/",
		checkAuthorised: checkAuthorised,
	})

	sort.Slice(g.prefixes, func(i, j int) bool {
		// Reverse string length sort so the longest path comes first
		return utf8.RuneCountInString(g.prefixes[j].path) < utf8.RuneCountInString(g.prefixes[i].path)
	})
}

func (g *Guard) RequireSignIn(path string) {
	g.Protect(path, g.isSignedIn)
}

func (g *Guard) RequireSignInPrefix(path string) {
	g.ProtectPrefix(path, g.isSignedIn)
}

func (g *Guard) Middleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// If a guard exists for an exact match on the path then we run that
		// otherwise we look for the longest matching prefix guard
		//
		// This way we guarantee that only the best matching guard will be run
		if checkAuthorised, ok := g.exact[r.URL.Path]; ok {
			ctx := r.Context()

			passport := g.svc.Passport(ctx)
			if err := checkAuthorised(passport); err != nil {
				if errors.Is(err, ErrRedirect) {
					g.svc.Sessions.Set(ctx, sess.Redirect, r.URL.String())

					http.Redirect(w, r, g.redirect(), http.StatusSeeOther)
				} else {
					g.svc.ErrorView(w, r, errors.Tracef(err), "error", nil)
				}

				return
			}
		} else {
			for _, guard := range g.prefixes {
				if !strings.HasPrefix(r.URL.Path, guard.path) {
					continue
				}

				ctx := r.Context()

				passport := g.svc.Passport(ctx)
				if err := guard.checkAuthorised(passport); err != nil {
					if errors.Is(err, ErrRedirect) {
						g.svc.Sessions.Set(ctx, sess.Redirect, r.URL.String())

						http.Redirect(w, r, g.redirect(), http.StatusSeeOther)
					} else {
						g.svc.ErrorView(w, r, errors.Tracef(err), "error", nil)
					}

					return
				}

				// We only want to apply the guard with the longest matching prefix
				// so we break out of the loop here to prevent running more guards
				// than we should
				break
			}
		}

		next(w, r)
	}
}
