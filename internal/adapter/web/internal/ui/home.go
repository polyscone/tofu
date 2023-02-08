package ui

import "net/http"

func (app *App) homeGet(w http.ResponseWriter, r *http.Request) {
	app.render(w, r, http.StatusOK, "home", nil)
}
