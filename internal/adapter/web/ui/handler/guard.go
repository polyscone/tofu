package handler

import (
	"net/http"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/polyscone/tofu/internal/adapter/web/passport"
	"github.com/polyscone/tofu/internal/adapter/web/sess"
	"github.com/polyscone/tofu/internal/app"
	"github.com/polyscone/tofu/internal/pkg/errors"
)

var ErrRedirect = errors.New("redirect")

type CheckFunc func(p passport.Passport) error
type PredicateFunc func(p passport.Passport) bool
type RedirectFunc func() string

type prefixGuard struct {
	path  string
	check CheckFunc
}

type Guard struct {
	h        *Handler
	exact    map[string]CheckFunc
	prefixes []prefixGuard
	redirect RedirectFunc
}

func NewGuard(h *Handler, redirect RedirectFunc) *Guard {
	return &Guard{
		h:        h,
		redirect: redirect,
		exact:    make(map[string]CheckFunc),
	}
}

func (g *Guard) isSignedIn(p passport.Passport) error {
	if !p.IsSignedIn() {
		return errors.Tracef(ErrRedirect)
	}

	return nil
}

func (g *Guard) isAuthorised(isAuthorised PredicateFunc) CheckFunc {
	return func(p passport.Passport) error {
		if !isAuthorised(p) {
			return app.ErrUnauthorised
		}

		return nil
	}
}

func (g *Guard) Protect(path string, check CheckFunc) {
	isPrefix := strings.HasSuffix(path, "/")
	path = strings.TrimSuffix(path, "/")

	g.exact[path] = check

	if isPrefix {
		g.prefixes = append(g.prefixes, prefixGuard{
			path:  path + "/",
			check: check,
		})

		sort.Slice(g.prefixes, func(i, j int) bool {
			// Make sure the shortest strings come first
			return utf8.RuneCountInString(g.prefixes[j].path) > utf8.RuneCountInString(g.prefixes[i].path)
		})
	}
}

func (g *Guard) RequireSignIn(pathNames ...string) {
	if len(pathNames) == 0 {
		g.Protect(g.h.mux.CurrentPrefix(), g.isSignedIn)
	} else {
		for _, name := range pathNames {
			g.Protect(g.h.mux.Path(name), g.isSignedIn)
		}
	}
}

func (g *Guard) RequireAuth(isAuthorised PredicateFunc, pathNames ...string) {
	if len(pathNames) == 0 {
		g.Protect(g.h.mux.CurrentPrefix(), g.isAuthorised(isAuthorised))
	} else {
		for _, name := range pathNames {
			g.Protect(g.h.mux.Path(name), g.isAuthorised(isAuthorised))
		}
	}
}

func (g *Guard) Middleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		for _, guard := range g.prefixes {
			if !strings.HasPrefix(r.URL.Path, guard.path) {
				continue
			}

			ctx := r.Context()

			passport := g.h.Passport(ctx)
			if err := guard.check(passport); err != nil {
				if errors.Is(err, ErrRedirect) {
					g.h.Sessions.Set(ctx, sess.Redirect, r.URL.String())

					http.Redirect(w, r, g.redirect(), http.StatusSeeOther)
				} else {
					g.h.ErrorView(w, r, errors.Tracef(err), "error", nil)
				}

				return
			}
		}

		if check, ok := g.exact[r.URL.Path]; ok {
			ctx := r.Context()

			passport := g.h.Passport(ctx)
			if err := check(passport); err != nil {
				if errors.Is(err, ErrRedirect) {
					g.h.Sessions.Set(ctx, sess.Redirect, r.URL.String())

					http.Redirect(w, r, g.redirect(), http.StatusSeeOther)
				} else {
					g.h.ErrorView(w, r, errors.Tracef(err), "error", nil)
				}

				return
			}
		}

		next(w, r)
	}
}
