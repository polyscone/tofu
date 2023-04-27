package ui

import "net/http"

func (ui *UI) homeGet(w http.ResponseWriter, r *http.Request) {
	ui.render(w, r, http.StatusOK, "home", nil)
}
