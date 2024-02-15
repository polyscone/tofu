// Pages
import "./pages/home.js"

// Account sign up
import "./pages/account/sign_up/form.js"
import "./pages/account/sign_up/success.js"
import "./pages/account/sign_up/verify.js"

// Account sign in
import "./pages/account/sign_in/web_form.js"
import "./pages/account/sign_in/recovery_code.js"
import "./pages/account/sign_in/totp.js"
import "./pages/account/sign_in/totp_required.js"

// Account reset password
import "./pages/account/reset_password/new_password.js"
import "./pages/account/reset_password/request.js"

// Errors
import "./pages/error/not_found.js"

platform.routes.protect(/^\/$/)

import Layout from "./master/layout.js"

function handle (opts) {
	opts ||= {}

	return {
		onmatch (args, requestedPath, route) {
			const isProtected = platform.routes.__protected.find(r => r.test(requestedPath || "/"))

			if (isProtected) {
				if (!platform.session.isSignedIn) {
					const redirect = requestedPath || "/"

					if (m.route.get() !== redirect) {
						platform.state.redirect = redirect
					}

					return m.route.set(platform.routes.path("account.sign_in"))
				}

				if (platform.session.isTOTPRequired) {
					return m.route.set(platform.routes.path("account.sign_in.totp_required"))
				}
			}

			if (typeof opts.onmatch === "function") {
				const res = opts.onmatch(args, requestedPath, route)

				if (res) {
					return res
				}
			}

			// Cancel redundant route resolutions
			if (m.route.get() === requestedPath) {
				return new Promise(() => {})
			}

			platform.modal.close()

			return opts.render
		},
		render (vnode) {
			return m(Layout, vnode)
		},
	}
}

export default function () {
	m.route.prefix = platform.config.prefix

	const routes = {}

	for (const pattern in platform.routes.__registered) {
		const route = platform.routes.__registered[pattern]

		routes[pattern] = handle({
			onmatch: route.onmatch,
			render: route.render,
		})
	}

	m.route(document.body, platform.routes.path("home"), routes)
}
