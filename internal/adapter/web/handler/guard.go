package handler

import (
	"fmt"
	"net/http"

	"github.com/polyscone/tofu/internal/adapter/web/passport"
	"github.com/polyscone/tofu/internal/adapter/web/sess"
)

type IsAuthorisedFunc func(passport passport.Passport) bool
type RedirectFunc func() string

type Guard struct {
	svc       *Services
	protected map[string]IsAuthorisedFunc
	redirect  RedirectFunc
}

func NewGuard(svc *Services, redirect RedirectFunc) *Guard {
	return &Guard{
		svc:       svc,
		redirect:  redirect,
		protected: make(map[string]IsAuthorisedFunc),
	}
}

func (g *Guard) ProtectFunc(path string, isAuthorised IsAuthorisedFunc) {
	if _, ok := g.protected[path]; ok {
		panic(fmt.Sprintf("a guard has already been registered for the path %q", path))
	}

	g.protected[path] = isAuthorised
}

func (g *Guard) Protect(path string) {
	g.ProtectFunc(path, func(passport passport.Passport) bool {
		return passport.IsAuthenticated()
	})
}

func (g *Guard) Middleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if isAuthorised, ok := g.protected[r.URL.Path]; ok {
			ctx := r.Context()

			passport := g.svc.Passport(ctx)
			if !isAuthorised(passport) {
				g.svc.Sessions.Set(ctx, sess.Redirect, r.URL.String())

				http.Redirect(w, r, g.redirect(), http.StatusSeeOther)

				return
			}
		}

		next(w, r)
	}
}
