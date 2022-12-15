package ui

import (
	"io/fs"
	"net/http"

	"github.com/polyscone/tofu/internal/adapter/web/internal/httputil"
	"github.com/polyscone/tofu/internal/pkg/errors"
)

func (app *App) faviconGet(w http.ResponseWriter, r *http.Request) {
	b, err := fs.ReadFile(app.files, "files/static/img/favicon.png")
	if err != nil {
		httputil.LogError(r, errors.Tracef(err))

		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)

		return
	}

	w.Header().Set("content-type", http.DetectContentType(b))
	w.Write(b)
}
