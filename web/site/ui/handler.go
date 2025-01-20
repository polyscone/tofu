package ui

import (
	"context"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"path"
	"slices"
	"strings"

	"github.com/polyscone/tofu/app"
	"github.com/polyscone/tofu/internal/httpx/middleware"
	"github.com/polyscone/tofu/internal/httpx/router"
	"github.com/polyscone/tofu/internal/i18n"
	"github.com/polyscone/tofu/web/guard"
	"github.com/polyscone/tofu/web/handler"
)

type PredicateFunc func(p guard.Passport) bool

type Handler struct {
	*handler.Handler
	signInPath         func() string
	mux                *router.ServeMux
	i18nRuntime        i18n.Runtime
	i18nRuntimeWrapper handler.WrapI18nRuntimeFunc
	Funcs              template.FuncMap
	HTML               *handler.Renderer
}

func NewHandler(base *handler.Handler, mux *router.ServeMux, signInPath func() string) *Handler {
	i18nRuntimeWrapper := handler.NewI18nRuntimeWrapper(mux)
	h := &Handler{
		Handler:            base,
		signInPath:         signInPath,
		mux:                mux,
		i18nRuntime:        i18nRuntimeWrapper(i18n.DefaultHTMLRuntime),
		i18nRuntimeWrapper: i18nRuntimeWrapper,
	}

	h.Funcs = handler.NewTemplateFuncs(template.FuncMap{
		"Path":          h.tmplPath,
		"HasPathPrefix": h.tmplHasPathPrefix,
	})

	templatePatterns := func(view string) []string {
		paths := []string{"partial/*.html"}
		fs.WalkDir(templateFiles, ".", func(p string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.IsDir() && p != "." && p != "component" && !strings.HasPrefix(p, "component/") {
				return fs.SkipDir
			}
			if path.Ext(p) != ".html" {
				return nil
			}

			dir := path.Dir(p)
			tp := dir + "/*.html"
			if !slices.Contains(paths, tp) {
				paths = append(paths, tp)
			}

			return nil
		})

		dir := path.Dir(view)

		paths = append(paths,
			"view/"+dir+"/com_*.html",
			"view/"+view+".html",
			"master/*.html",
		)

		return paths
	}

	h.HTML = handler.NewRenderer(handler.RendererConfig{
		Handler:          h.Handler,
		AssetTags:        AssetTags,
		AssetFiles:       AssetFiles,
		TemplateFiles:    templateFiles,
		TemplatePatterns: templatePatterns,
		Funcs:            h.Funcs,
		T:                h.T,
		WrapI18nRuntime:  i18nRuntimeWrapper,
		Process: func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("content-type", "text/html; charset=utf-8")
		},
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

func (h *Handler) T(ctx context.Context, message i18n.Message) string {
	locale := h.Locale(ctx)
	res, err := i18n.T(h.i18nRuntime, locale, message)
	if err != nil {
		logger := h.Logger(ctx)

		logger.Error("site handler: i18n T", "err", err)
	}

	return res.AsString().Value
}

func (h *Handler) SendEmail(ctx context.Context, from, to string, view string, vars handler.Vars) error {
	templatePatterns := func(name string) []string {
		return []string{"email/" + view + ".html"}
	}

	return h.Handler.SendEmail(ctx, templateFiles, templatePatterns, h.Funcs, h.i18nRuntimeWrapper, from, to, view, vars)
}

func (h *Handler) HasPathPrefix(value string, name string, paramArgPairs ...any) bool {
	p := h.Path(name, paramArgPairs...)
	p = strings.TrimSuffix(p, "/")

	return value == p || strings.HasPrefix(value, p+"/")
}

func (h *Handler) Path(name string, paramArgPairs ...any) string {
	return h.mux.Path(name, paramArgPairs...)
}

func (h *Handler) PathQuery(r *http.Request, name string, paramArgPairs ...any) string {
	q := r.URL.Query().Encode()
	if q != "" && !strings.HasPrefix(q, "?") {
		q = "?" + q
	}

	return h.Path(name, paramArgPairs...) + q
}

func (h *Handler) AddFlashf(ctx context.Context, message i18n.Message) {
	flash := h.Session.Flash(ctx)

	flash = append(flash, h.T(ctx, message))

	h.Session.SetFlash(ctx, flash)
}

func (h *Handler) AddFlashWarningf(ctx context.Context, message i18n.Message) {
	flash := h.Session.FlashWarning(ctx)

	flash = append(flash, h.T(ctx, message))

	h.Session.SetFlashWarning(ctx, flash)
}

func (h *Handler) AddFlashImportantf(ctx context.Context, message i18n.Message) {
	flash := h.Session.FlashImportant(ctx)

	flash = append(flash, h.T(ctx, message))

	h.Session.SetFlashImportant(ctx, flash)
}

func (h *Handler) AddFlashErrorf(ctx context.Context, message i18n.Message) {
	flash := h.Session.FlashError(ctx)

	flash = append(flash, h.T(ctx, message))

	h.Session.SetFlashError(ctx, flash)
}

func (h *Handler) RequireSignIn(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		isSignedIn := h.Session.IsSignedIn(ctx)

		if !isSignedIn {
			h.Session.SetRedirect(ctx, r.URL.String())

			http.Redirect(w, r, h.signInPath(), http.StatusSeeOther)

			return
		}

		next(w, r)
	}
}

func (h *Handler) RequireSignInIf(check PredicateFunc) middleware.Middleware {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			isSignedIn := h.Session.IsSignedIn(ctx)
			passport := h.Passport(ctx)

			if !isSignedIn && check(passport) {
				h.Session.SetRedirect(ctx, r.URL.String())

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
				h.HTML.ErrorView(w, r, "require auth", app.ErrForbidden, "error", nil)

				return
			}

			next(w, r)
		}
	}
}
