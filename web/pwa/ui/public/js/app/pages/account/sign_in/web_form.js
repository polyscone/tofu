import { ErrorBanner } from "../../../components/error.js"
import { EmailInput, PasswordInput, TokenInput } from "../../../components/forms.js"

function SignInWithGoogle () {
	let rendered = false

	function renderButton (vnode) {
		if (window.google && !rendered) {
			const node = vnode.dom.parentNode.querySelector("#sign-in__gsi_button")

			google.accounts.id.renderButton(node, {
				type: "standard",
				shape: "rectangle",
				theme: "outline",
				text: "signin_with",
				size: "large",
				logo_alignment: "center",
			})

			rendered = true
		}
	}

	function signIn (jwt) {
		platform.loading(async () => {
			const res = await platform.api.account.signInWithGoogle(jwt)

			if (res.ok) {
				if (platform.session.isAwaitingTOTP) {
					m.route.set(platform.routes.path("account.sign_in.totp"))
				} else if (!res.body.isSignedIn) {
					// Signing in with Google can trigger a sign up
					//
					// If that's the case, and the response was ok, then if they aren't
					// already signed in then it's likely to mean that their account
					// was created but requires manual activation by an admin through
					// user management, so we show them the sign up success page
					m.route.set(platform.routes.path("account.sign_up.success"))
				} else {
					platform.tryRedirect(platform.routes.path("home"))
				}
			}
		})
	}

	return {
		oncreate (vnode) {
			if (platform.config.googleSignInEnabled && platform.config.googleSignInClientId && !window.gsiLoaded) {
				window.gsiLoaded = true

				const s = document.createElement("script")

				s.src = "https://accounts.google.com/gsi/client"
				s.defer = true
				s.onload = function () {
					google.accounts.id.initialize({
						client_id: platform.config.googleSignInClientId,
						context: "signin",
						ux_mode: "popup",
						async callback (res) {
							await signIn(res.credential)

							m.redraw()
						},
					})

					m.redraw()
				}

				document.body.appendChild(s)
			}

			renderButton(vnode)
		},
		onupdate: renderButton,
		onremove () {
			rendered = false
		},
		view () {
			if (!platform.config.googleSignInEnabled || !window.google) {
				return null
			}

			return [
				m(".google-signin__dummy"),
				m("#sign-in__gsi_button.g_id_signin"),
			]
		},
	}
}

function SignInWithFacebook () {
	let rendered = false

	function renderButton (vnode) {
		if (window.FB && !rendered) {
			const parent = vnode.dom.parentNode

			FB.XFBML.parse(parent)

			rendered = true
		}
	}

	function signIn () {
		platform.loading(new Promise(resolve => {
			FB.getLoginStatus(response => {
				if (response.status !== "connected") {
					console.error("the user either did not sign in to Facebook or did not authorize the app", response)

					resolve()

					return
				}

				if (!response.authResponse) {
					console.error("no auth response", response)

					resolve()

					return
				}

				const userId = response.authResponse.userID
				const accessToken = response.authResponse.accessToken

				FB.api("/me?fields=email", async response => {
					const email = response.email
					const res = await platform.api.account.signInWithFacebook(userId, accessToken, email)

					if (res.ok) {
						if (platform.session.isAwaitingTOTP) {
							m.route.set(platform.routes.path("account.sign_in.totp"))
						} else if (!res.body.isSignedIn) {
							// Signing in with Facebook can trigger a sign up
							//
							// If that's the case, and the response was ok, then if they aren't
							// already signed in then it's likely to mean that their account
							// was created but requires manual activation by an admin through
							// user management, so we show them the sign up success page
							m.route.set(platform.routes.path("account.sign_up.success"))
						} else {
							platform.tryRedirect(platform.routes.path("home"))
						}
					}

					resolve()
				})
			})
		}))
	}

	window.onFacebookSignIn = signIn

	return {
		oncreate (vnode) {
			if (platform.config.facebookSignInEnabled && platform.config.facebookSignInAppId && !window.FB) {
				const s = document.createElement("script")

				s.src = `https://connect.facebook.net/en_GB/sdk.js#xfbml=1&version=v18.0&appId=${platform.config.facebookSignInAppId}`
				s.crossorigin = "anonymous"
				s.nonce = "PewzUoD4"
				s.defer = true
				s.onload = function () {
					m.redraw()
				}

				document.body.appendChild(s)
			}

			renderButton(vnode)
		},
		onupdate: renderButton,
		onremove () {
			rendered = false
		},
		view () {
			if (!platform.config.facebookSignInEnabled || !window.FB) {
				return null
			}

			return [
				m("#fb-root"),
				m(".fb-login-button", {
					"data-size": "large",
					"data-button-type": "continue_with",
					"data-layout": "default",
					"data-auto-logout-link": "false",
					"data-use-continue-as": "false",
					"data-scope": "email",
					"data-onlogin": "onFacebookSignIn",
				})
			]
		},
	}
}

function SignInMagicLink () {
	const state = {
		error: "",
		errors: {},
		token: "",
	}

	function signIn (e) {
		e.preventDefault()

		platform.loading(async () => {
			const res = await platform.api.account.signInWithMagicLink(state.token)

			state.error = res.body.error || ""
			state.errors = res.body.fields || {}

			if (res.ok) {
				if (platform.session.isAwaitingTOTP) {
					m.route.set(platform.routes.path("account.sign_in.totp"))
				} else if (!res.body.isSignedIn) {
					// Signing in with a magic link can trigger a sign up
					//
					// If that's the case, and the response was ok, then if they aren't
					// already signed in then it's likely to mean that their account
					// was created but requires manual activation by an admin through
					// user management, so we show them the sign up success page
					m.route.set(platform.routes.path("account.sign_up.success"))
				} else {
					platform.tryRedirect(platform.routes.path("home"))
				}
			}
		})
	}

	return {
		oncreate () {
			if (platform.session.isSignedIn || !platform.config.magicLinkSignInEnabled) {
				m.route.set(platform.routes.path("account.sign_in"))
			}
		},
		view () {
			return [
				m("h1", "{{.T "pwa.account.sign_in.magic_link.title"}}"),
				m("p", "{{.T "pwa.account.sign_in.magic_link.text"}}"),
				m("form", { onsubmit: signIn }, [
					state.error ? m(ErrorBanner, state.error) : null,
					state.error ? m("p", "{{.T "pwa.account.sign_in.magic_link.code_expired_text"}}") : null,
					m(TokenInput, {
						label: "{{.T "pwa.account.sign_in.magic_link.code_label"}}",
						name: "token",
						required: true,
						error: state.errors.token,
						oninput (e) { state.token = e.target.value },
					}),
					m("button[type=submit]", "{{.T "pwa.account.sign_in.magic_link.sign_in_button"}}"),
					state.error ? m(m.route.Link, { href: platform.routes.path("account.sign_in.magic_link.request"), class: "btn btn--alt" }, "{{.T "pwa.account.sign_in.magic_link.request_new_code_button"}}") : null,
				]),
			]
		},
	}
}

function SignInMagicLinkRequest () {
	const state = {
		error: "",
		errors: {},
		email: "",
	}

	function signIn (e) {
		e.preventDefault()

		platform.loading(async () => {
			const res = await platform.api.account.requestSignInMagicLink(state.email)

			state.error = res.body.error || ""
			state.errors = res.body.fields || {}

			if (res.ok) {
				m.route.set(platform.routes.path("account.sign_in.magic_link"))
			}
		})
	}

	return {
		oncreate () {
			if (platform.session.isSignedIn || !platform.config.magicLinkSignInEnabled) {
				m.route.set(platform.routes.path("account.sign_in"))
			}
		},
		view () {
			return [
				m("h1", "{{.T "pwa.account.sign_in.magic_link_request.title"}}"),
				m("form", { onsubmit: signIn }, [
					state.error ? m(ErrorBanner, state.error) : null,
					m(EmailInput, {
						label: "{{.T "pwa.account.sign_in.magic_link_request.email_label"}}",
						name: "email",
						required: true,
						autocomplete: "username",
						error: state.errors.email,
						oninput (e) { state.email = e.target.value },
					}),
					m("button[type=submit]", "{{.T "pwa.account.sign_in.magic_link_request.send_code_button"}}"),
					m("div.sign-in-alt", [
						m("p.sign-in-alt__title", "{{.T "pwa.account.sign_in.magic_link_request.alt_sign_in_title"}}"),
						m(m.route.Link, { href: platform.routes.path("account.sign_in"), class: "btn btn--alt btn--large" }, "{{.T "pwa.account.sign_in.magic_link_request.sign_in_with_password_button"}}"),
						m(SignInWithGoogle),
						m(SignInWithFacebook),
					]),
				]),
			]
		},
	}
}

function SignIn () {
	const state = {
		error: "",
		errors: {},
		email: "",
		password: "",
	}

	function signOut (e) {
		e.preventDefault()

		platform.loading(platform.api.account.signOut)
	}

	function signIn (e) {
		e.preventDefault()

		platform.loading(async () => {
			const res = await platform.api.account.signInWithPassword(state.email, state.password)

			state.error = res.body.error || ""
			state.errors = res.body.fields || {}

			if (res.ok) {
				if (platform.session.isAwaitingTOTP) {
					m.route.set(platform.routes.path("account.sign_in.totp"))
				} else {
					platform.tryRedirect(platform.routes.path("home"))
				}
			} else if (res.status === platform.http.tooManyRequests) {
				let { inLast, unlockIn } = res.body

				if (unlockIn) {
					unlockIn = ` in ${unlockIn}`
				}

				state.error = `{{.T "pwa:account.sign_in.throttled.error"}}`
			}
		})
	}

	return {
		view () {
			if (platform.session.isSignedIn) {
				return [
					m("p", "You're already signed in."),
					m("form", { onsubmit: signOut }, m(".bag", [
						m("button[type=submit].btn--link", "{{.T "pwa.account.sign_in.already_signed_in.sign_out_button"}}"),
					])),
				]
			}

			return [
				m("h1", "{{.T "pwa.account.sign_in.form.title"}}"),
				m("form", { onsubmit: signIn }, [
					state.error ? m(ErrorBanner, state.error) : null,
					m(EmailInput, {
						label: "{{.T "pwa.account.sign_in.form.email_label"}}",
						name: "email",
						required: true,
						autocomplete: "username",
						error: state.errors.email,
						oninput (e) { state.email = e.target.value },
					}),
					m(PasswordInput, {
						label: "{{.T "pwa.account.sign_in.form.password_label"}}",
						name: "password",
						required: true,
						autocomplete: "current-password",
						error: state.errors.password,
						oninput (e) { state.password = e.target.value },
					}),
					m("button[type=submit]", "{{.T "pwa.account.sign_in.form.sign_in_button"}}"),
					platform.config.signUpEnabled ? m(m.route.Link, { href: platform.routes.path("account.sign_up") }, "{{.T "pwa.account.sign_in.form.sign_up_button"}}") : null,
					m(m.route.Link, { href: platform.routes.path("account.reset_password") }, "{{.T "pwa.account.sign_in.form.reset_password_button"}}"),
					m("div.sign-in-alt", [
						m("p.sign-in-alt__title", "{{.T "pwa.account.sign_in.form.alt_sign_in_title"}}"),
						platform.config.magicLinkSignInEnabled ? m(m.route.Link, { href: platform.routes.path("account.sign_in.magic_link.request"), class: "btn btn--alt btn--large" }, "{{.T "pwa.account.sign_in.form.sign_in_with_magic_link_button"}}") : null,
						m(SignInWithGoogle),
						m(SignInWithFacebook),
					]),
				]),
			]
		},
	}
}

let isFirstLoad = true

platform.routes.register("/account/sign-in", {
	name: "account.sign_in",
	onmatch () {
		if (platform.config.magicLinkSignInEnabled && isFirstLoad) {
			isFirstLoad = false

			return m.route.set(platform.routes.path("account.sign_in.magic_link.request"))
		}
	},
	render: SignIn,
})

platform.routes.register("/account/sign-in/magic-link", {
	name: "account.sign_in.magic_link",
	render: SignInMagicLink,
})

platform.routes.register("/account/sign-in/magic-link/request", {
	name: "account.sign_in.magic_link.request",
	onmatch () {
		isFirstLoad = false
	},
	render: SignInMagicLinkRequest,
})
