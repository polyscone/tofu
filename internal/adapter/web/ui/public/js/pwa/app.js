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

window.addEventListener("load", async () => {
	// Although navigator.onLine is only reliable when its value is false
	// we set it here anyway just to make sure we have an initial value whilst
	// we wait for an API call to verify actually connectivity
	app.isOnline = navigator.onLine

	await app.api.meta.updateOnlineStatus()
	await app.api.account.pollSession()
	await app.api.security.updateCSRFToken()

	m.redraw()
})

window.addEventListener("online", async () => {
	await app.api.meta.updateOnlineStatus()

	m.redraw()
})

window.addEventListener("offline", async () => {
	// Going offline is the only time navigator.onLine is reliable, so we
	// can just set the value directly instead of waiting for an API call
	app.isOnline = navigator.onLine

	m.redraw()
})

router()

if ("serviceWorker" in navigator) {
	navigator.serviceWorker.register("/pwa_service_worker.js")
}
