package ui

import "net/http"

func (app *App) accountLoginGet(w http.ResponseWriter, r *http.Request) {
	app.render(w, r, http.StatusOK, "account_login", nil)
}
