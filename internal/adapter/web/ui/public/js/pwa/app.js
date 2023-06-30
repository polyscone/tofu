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
}

router()

;(async () => {
	await app.api.account.updateSession()

	m.redraw()
})()
