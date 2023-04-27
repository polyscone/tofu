package ui

import "net/http"

func (ui *UI) accountGet(w http.ResponseWriter, r *http.Request) {
	ui.render(w, r, http.StatusOK, "account", nil)
}
