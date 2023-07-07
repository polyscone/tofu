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
		badGateway: 502,
	},
	network: "connected", // "offline" | "disconnected" | "connected"
}

window.addEventListener("load", async () => {
	await app.api.meta.pollNetworkStatus()
	await app.api.account.pollSession()
	await app.api.security.updateCSRFToken()

	m.redraw()
})

window.addEventListener("online", app.api.meta.pollNetworkStatus)
window.addEventListener("offline", app.api.meta.pollNetworkStatus)

router()

if ("serviceWorker" in navigator) {
	navigator.serviceWorker.register("/pwa_service_worker.js")
}
