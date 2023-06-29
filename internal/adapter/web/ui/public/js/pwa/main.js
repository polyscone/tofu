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
		next: "",
		set isSignedIn (value) {
			sessionStorage.setItem("isSignedIn", JSON.stringify(value))
		},
		get isSignedIn () {
			const value = sessionStorage.getItem("isSignedIn")

			if (!value) {
				return false
			}

			return JSON.parse(value)
		},
		set isAwaitingTOTP (value) {
			sessionStorage.setItem("isAwaitingTOTP", JSON.stringify(value))
		},
		get isAwaitingTOTP () {
			const value = sessionStorage.getItem("isAwaitingTOTP")

			if (!value) {
				return false
			}

			return JSON.parse(value)
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

			if (app.auth.next === requestedPath) {
				app.auth.next = ""
			}

			let redirect = ""
			const signInPath = "/sign-in"
			const signInTOTPPath = "/sign-in/totp"
			const isSignInRoute = [signInPath, signInTOTPPath].includes(requestedPath)

			if (!app.auth.isSignedIn && !isSignInRoute) {
				app.auth.next = requestedPath
				redirect = signInPath
			}

			if (!app.auth.isAwaitingTOTP && requestedPath === signInTOTPPath) {
				redirect = signInPath
			}

			if (redirect) {
				m.route.set(redirect)
			}
		},
		render: () => m(Layout, m(component)),
	}
}
