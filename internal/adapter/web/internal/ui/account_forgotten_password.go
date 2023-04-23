package ui

import "net/http"

func (app *App) accountForgottenPasswordGet(w http.ResponseWriter, r *http.Request) {
	app.render(w, r, http.StatusOK, "account_forgotten_password", nil)
}
