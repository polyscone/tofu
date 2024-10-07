package web

import (
	"bytes"
	"embed"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"mime"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/polyscone/tofu/app"
	"github.com/polyscone/tofu/cache"
	"github.com/polyscone/tofu/errsx"
	"github.com/polyscone/tofu/fsx"
	"github.com/polyscone/tofu/httpx"
	"github.com/polyscone/tofu/httpx/router"
	"github.com/polyscone/tofu/web/handler"
)

//go:embed "all:ui/public"
var uiFiles embed.FS

const uiPublicDir = "ui/public"

var uiPublicFiles = fsx.NewStack(fsx.RelDirFS(uiPublicDir), errsx.Must(fs.Sub(uiFiles, uiPublicDir)))

// HandlerTimeout should be used as the value in all timeout middleware, and as the
// base value to calculate http.Server timeouts from.
const HandlerTimeout = 5 * time.Second

var muxes = cache.New[string, *http.ServeMux]()

func NewRouter(tenant *handler.Tenant) http.Handler {
	key := tenant.Key + "." + tenant.Kind

	return muxes.LoadOrStore(key, func() *http.ServeMux {
		mux := http.NewServeMux()
		h := handler.New(tenant)

		switch tenant.Kind {
		case "site":
			mux.Handle("/", NewSiteRouter(h))

		case "pwa":
			pwa := NewPWARouter(h)
			apiV1 := NewAPIRouterV1(h)

			mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				if strings.HasPrefix(r.URL.Path, app.BasePath+"/api/") {
					apiV1.ServeHTTP(w, r)

					return
				}

				pwa.ServeHTTP(w, r)
			})
		}

		return mux
	})
}

var ErrNoIndex = errors.New("no index file")

type FileServerErrorHandler func(w http.ResponseWriter, r *http.Request, err error)

func newFileServer(fsys fs.FS, basePath string, mux *router.ServeMux, renderer *handler.Renderer, errorHandler FileServerErrorHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if allowedMethods, notAllowed := httpx.MethodNotAllowed(mux, r); notAllowed {
			w.Header().Set("allow", strings.Join(allowedMethods, ", "))

			errorHandler(w, r, httpx.ErrMethodNotAllowed)

			return
		}

		upath := r.URL.Path
		if !strings.HasPrefix(upath, "/") {
			upath = "/" + upath
			r.URL.Path = upath
		}
		if basePath != "" {
			upath = strings.TrimPrefix(upath, basePath)
		}
		upath = path.Clean(upath)

		fpath := strings.TrimPrefix(upath, "/")
		f, err := fsys.Open(fpath)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) || errors.Is(err, fs.ErrInvalid) {
				errorHandler(w, r, fmt.Errorf("%w: %w", httpx.ErrNotFound, err))
			} else {
				errorHandler(w, r, fmt.Errorf("%w: %w", httpx.ErrInternalServerError, err))
			}

			return
		}
		defer f.Close()

		stat, err := f.Stat()
		if err != nil {
			errorHandler(w, r, fmt.Errorf("%w: stat: %w", httpx.ErrInternalServerError, err))

			return
		}

		// If the directory has either an index.html or index.htm file then display that
		// otherwise we forbid viewing directories
		if stat.IsDir() {
			if !strings.HasSuffix(fpath, "/") {
				fpath += "/"
			}
			if !strings.HasSuffix(upath, "/") {
				upath += "/"
			}

			var hasIndex bool
			for _, name := range []string{"index.html", "index.htm"} {
				if _f, err := fsys.Open(fpath + name); err == nil {
					defer _f.Close()

					_stat, err := _f.Stat()
					if err != nil {
						errorHandler(w, r, fmt.Errorf("%w: stat: %w", httpx.ErrInternalServerError, err))

						return
					}

					f = _f
					stat = _stat
					fpath += name
					upath += name
					hasIndex = true

					break
				}
			}

			if !hasIndex {
				errorHandler(w, r, fmt.Errorf("%w: directory: %w", httpx.ErrForbidden, ErrNoIndex))

				return
			}
		}

		b, err := io.ReadAll(f)
		if err != nil {
			errorHandler(w, r, fmt.Errorf("%w: read all: %w", httpx.ErrInternalServerError, err))

			return
		}

		modtime := stat.ModTime()
		if bytes.Contains(b, []byte("{{")) && bytes.Contains(b, []byte("}}")) {
			modtime = time.Time{}

			contentType := mime.TypeByExtension(path.Ext(upath))
			if contentType == "" {
				contentType = http.DetectContentType(b)
			}
			mediaType, _, _ := mime.ParseMediaType(contentType)
			if mediaType == "" {
				mediaType = "application/octet-stream"
			}

			render := renderer.Text
			if mediaType == "text/html" {
				render = renderer.HTML
			}

			var buf bytes.Buffer
			if err := render(&buf, r, http.StatusOK, string(b), nil); err != nil {
				errorHandler(w, r, fmt.Errorf("%w: render: %w", httpx.ErrInternalServerError, err))

				return
			}

			b = buf.Bytes()
		}

		http.ServeContent(w, r, stat.Name(), modtime, bytes.NewReader(b))
	}
}
