package ui

import "net/http"

func (app *App) robotsGet(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("content-type", "text/plain")
	w.Write([]byte("User-agent: *\nDisallow: /\n"))
}
