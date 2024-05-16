package ui

import (
	"bytes"
	"context"
	"embed"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"path"
	"strings"

	"github.com/polyscone/tofu/app"
	"github.com/polyscone/tofu/dev"
	"github.com/polyscone/tofu/errsx"
	"github.com/polyscone/tofu/fstack"
	"github.com/polyscone/tofu/httpx/middleware"
	"github.com/polyscone/tofu/httpx/router"
	"github.com/polyscone/tofu/web/guard"
	"github.com/polyscone/tofu/web/handler"
	"github.com/polyscone/tofu/web/sess"
)

//go:embed "all:template"
var files embed.FS

const templateDir = "template"

var templateFiles = fstack.New(dev.RelDirFS(templateDir), errsx.Must(fs.Sub(files, templateDir)))

type Handler struct {
	*handler.Handler
	signInPath func() string
	baseURL    string
	mux        *router.ServeMux
	funcs      template.FuncMap
	Plain      *handler.Renderer
	HTML       *handler.Renderer
	JSON       *handler.Renderer
}

func NewHandler(base *handler.Handler, mux *router.ServeMux, baseURL string, signInPath func() string) *Handler {
	h := &Handler{
		Handler:    base,
		signInPath: signInPath,
		baseURL:    baseURL,
		mux:        mux,
	}

	h.funcs = handler.NewTemplateFuncs(template.FuncMap{
		"Path":          h.tmplPath,
		"HasPathPrefix": h.tmplHasPathPrefix,
	})

	templatePaths := func(view string) []string {
		dir := path.Dir(view)

		return []string{
			"partial/*.html",
			"view/" + dir + "/com_*.html",
			"view/" + view + ".html",
			"master/*.html",
		}
	}

	h.Plain = handler.NewRenderer(h.Handler, templateFiles, templatePaths, h.funcs, func(w http.ResponseWriter, r *http.Request, template *bytes.Buffer) []byte {
		w.Header().Set("content-type", "text/plain; charset=utf-8")

		return nil
	})

	h.HTML = handler.NewRenderer(h.Handler, templateFiles, templatePaths, h.funcs, func(w http.ResponseWriter, r *http.Request, template *bytes.Buffer) []byte {
		w.Header().Set("content-type", "text/html; charset=utf-8")

		return nil
	})

	h.JSON = handler.NewRenderer(h.Handler, templateFiles, templatePaths, h.funcs, func(w http.ResponseWriter, r *http.Request, template *bytes.Buffer) []byte {
		w.Header().Set("content-type", "application/json")

		return nil
	})

	return h
}

func (h *Handler) tmplPath(name string, paramArgPairs ...any) template.URL {
	return template.URL(h.Path(name, paramArgPairs...))
}

func (h *Handler) tmplHasPathPrefix(value any, name string, paramArgPairs ...any) bool {
	v := fmt.Sprintf("%v", value)
	p := h.Path(name, paramArgPairs...)
	p = strings.TrimSuffix(p, "/")

	return v == p || strings.HasPrefix(v, p+"/")
}

func (h *Handler) SendEmail(ctx context.Context, from, to string, view string, vars handler.Vars) error {
	templatePaths := []string{"email/" + view + ".html"}

	return h.Handler.SendEmail(ctx, templateFiles, templatePaths, h.funcs, from, to, view, vars)
}

func (h *Handler) HasPathPrefix(value string, name string, paramArgPairs ...any) bool {
	p := h.Path(name, paramArgPairs...)
	p = strings.TrimSuffix(p, "/")

	return value == p || strings.HasPrefix(value, p+"/")
}

func (h *Handler) Path(name string, paramArgPairs ...any) string {
	p := h.mux.Path(name, paramArgPairs...)
	if h.baseURL != "" && !strings.HasSuffix(p, h.baseURL) {
		p = strings.TrimSuffix(h.baseURL+p, "/")
	}

	return p
}

func (h *Handler) PathQuery(r *http.Request, name string, paramArgPairs ...any) string {
	q := r.URL.Query().Encode()
	if q != "" && !strings.HasPrefix(q, "?") {
		q = "?" + q
	}

	return h.Path(name, paramArgPairs...) + q
}

func (h *Handler) AddFlashf(ctx context.Context, format string, a ...any) {
	flash := h.Sessions.GetStrings(ctx, sess.Flash)

	flash = append(flash, fmt.Sprintf(format, a...))

	h.Sessions.Set(ctx, sess.Flash, flash)
}

func (h *Handler) AddFlashImportantf(ctx context.Context, format string, a ...any) {
	flash := h.Sessions.GetStrings(ctx, sess.FlashImportant)

	flash = append(flash, fmt.Sprintf(format, a...))

	h.Sessions.Set(ctx, sess.FlashImportant, flash)
}

func (h *Handler) AddFlashErrorf(ctx context.Context, format string, a ...any) {
	flash := h.Sessions.GetStrings(ctx, sess.FlashError)

	flash = append(flash, fmt.Sprintf(format, a...))

	h.Sessions.Set(ctx, sess.FlashError, flash)
}

func (h *Handler) RequireSignIn(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		isSignedIn := h.Sessions.GetBool(ctx, sess.IsSignedIn)

		if !isSignedIn {
			h.Sessions.Set(ctx, sess.Redirect, r.URL.String())

			http.Redirect(w, r, h.signInPath(), http.StatusSeeOther)

			return
		}

		next(w, r)
	}
}

type PredicateFunc func(p guard.Passport) bool

func (h *Handler) RequireSignInIf(check PredicateFunc) middleware.Middleware {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			isSignedIn := h.Sessions.GetBool(ctx, sess.IsSignedIn)
			passport := h.Passport(ctx)

			if !isSignedIn && check(passport) {
				h.Sessions.Set(ctx, sess.Redirect, r.URL.String())

				http.Redirect(w, r, h.signInPath(), http.StatusSeeOther)

				return
			}

			next(w, r)
		}
	}
}

func (h *Handler) CanAccess(check PredicateFunc) middleware.Middleware {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			passport := h.Passport(ctx)

			if !check(passport) {
				h.HTML.ErrorView(w, r, "require auth", app.ErrForbidden, "site/error", nil)

				return
			}

			next(w, r)
		}
	}
}
