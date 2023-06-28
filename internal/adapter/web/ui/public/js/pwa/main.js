import Layout from "./layout.js"
import Home from "./home.js"
import SignIn from "./sign_in.js"
import NotFound from "./not_found.js"
import api from "./api.js"
import { show, hide } from "./loading.js"

window.app = {
	name: appData.Name,
	description: appData.Description,
	auth: {
		set isSignedIn (value) {
			sessionStorage.setItem("isSignedIn", JSON.stringify(value))
		},
		get isSignedIn () {
			return JSON.parse(sessionStorage.getItem("isSignedIn"))
		},
		redirect: "",
	},
	api,
	loading: {
		show,
		hide,
	},
	http: {
		noContent: 204,
		tooManyRequests: 429,
	}
}

m.route.prefix = ""

m.route(document.body, "/", {
	"/": handle(Home),
	"/sign-in": handle(SignIn),
	"/:rest...": handle(NotFound),
})

function handle (component) {
	return {
		onmatch (args, requestedPath) {
			if (m.route.get() === requestedPath) {
				return new Promise(() => {})
			}

			if (app.auth.redirect === requestedPath) {
				app.auth.redirect = ""
			}

			const signInPath = "/sign-in"

			if (!app.auth.isSignedIn && requestedPath !== signInPath) {
				app.auth.redirect = requestedPath

				m.route.set(signInPath)
			}
		},
		render: () => m(Layout, m(component)),
	}
}
