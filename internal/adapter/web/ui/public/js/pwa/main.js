import Layout from "./layout.js"
import Home from "./home.js"
import { SignInPassword, SignInTOTP } from "./sign_in.js"
import NotFound from "./not_found.js"
import api from "./api.js"
import { show, hide } from "./loading.js"

window.app = {
	name: appData.Name,
	description: appData.Description,
	auth: {
		redirect: "",
		set isSignedIn (value) {
			sessionStorage.setItem("isSignedIn", JSON.stringify(value))
		},
		get isSignedIn () {
			return JSON.parse(sessionStorage.getItem("isSignedIn"))
		},
		set isAwaitingTOTP (value) {
			sessionStorage.setItem("isAwaitingTOTP", JSON.stringify(value))
		},
		get isAwaitingTOTP () {
			return JSON.parse(sessionStorage.getItem("isAwaitingTOTP"))
		},
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
	"/sign-in": handle(SignInPassword),
	"/sign-in/totp": handle(SignInTOTP),
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
			const signInTOTPPath = "/sign-in/totp"

			if (!app.auth.isSignedIn && !app.auth.isAwaitingTOTP && requestedPath !== signInPath) {
				app.auth.redirect = requestedPath

				m.route.set(signInPath)
			}

			if (app.auth.isAwaitingTOTP && requestedPath !== signInTOTPPath) {
				app.auth.redirect = requestedPath

				m.route.set(signInTOTPPath)
			}
		},
		render: () => m(Layout, m(component)),
	}
}
