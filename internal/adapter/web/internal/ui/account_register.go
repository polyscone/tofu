package ui

import "net/http"

func (app *App) accountRegisterGet(w http.ResponseWriter, r *http.Request) {
	app.render(w, r, http.StatusOK, "account_register", nil)
}
