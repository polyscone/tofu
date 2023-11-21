// Pages
import "./pages/home.js"

// Account sign up
import "./pages/account/sign_up/form.js"
import "./pages/account/sign_up/success.js"
import "./pages/account/sign_up/verify.js"

// Account sign in
import "./pages/account/sign_in/password.js"
import "./pages/account/sign_in/recovery_code.js"
import "./pages/account/sign_in/totp.js"
import "./pages/account/sign_in/totp_required.js"

// Account reset password
import "./pages/account/reset_password/new_password.js"
import "./pages/account/reset_password/request.js"

// Errors
import "./pages/error/not_found.js"

import Layout from "./master/layout.js"

function handle (component, opts) {
	opts ||= {}

	return {
		onmatch (args, requestedPath) {
			if (opts.requireSignIn) {
				if (!platform.session.isSignedIn) {
					const redirect = requestedPath || "/"

					if (m.route.get() !== redirect) {
						platform.state.redirect = redirect
					}

					return m.route.set(platform.routes.accountSignIn.pattern)
				}

				if (platform.session.isTOTPRequired) {
					return m.route.set(platform.routes.accountSignInTOTPRequired.pattern)
				}
			}

			if (typeof opts.onmatch === "function") {
				const res = opts.onmatch(args, requestedPath)

				if (res) {
					return res
				}
			}

			// Cancel redundant route resolutions
			if (m.route.get() === requestedPath) {
				return new Promise(() => {})
			}

			return component
		},
		render (vnode) {
			return m(Layout, vnode)
		},
	}
}

export default function () {
	m.route.prefix = platform.config.prefix

	const routes = {}

	for (const key in platform.routes) {
		const route = platform.routes[key]

		routes[route.pattern] = handle(route.component, {
			onmatch: route.onmatch,
			requireSignIn: route.requireSignIn || false,
		})
	}

	m.route(document.body, platform.routes.home.pattern, routes)
}
