import api from "./api.js"
import router from "./router.js"
import { show, hide } from "./loading.js"

window.app = {
	get session () {
		return api.account.session()
	},
	api,
	loading: {
		show,
		hide,
	},
	http: {
		noContent: 204,
		tooManyRequests: 429,
	},
	isOnline: navigator.onLine,
}

router()

;(async () => {
	await app.api.account.updateSession()
	await app.api.security.updateCSRFToken()

	m.redraw()
})()

window.addEventListener("load", () => {
	app.isOnline = navigator.onLine

	m.redraw()
})

window.addEventListener("online", () => {
	app.isOnline = navigator.onLine

	m.redraw()
})

window.addEventListener("offline", () => {
	app.isOnline = navigator.onLine

	m.redraw()
})

if ("serviceWorker" in navigator) {
	navigator.serviceWorker.register("/pwa_service_worker.js")
}
