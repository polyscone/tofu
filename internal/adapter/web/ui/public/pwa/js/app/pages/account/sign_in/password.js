import { ErrorBanner } from "../../../components/error.js"
import { EmailInput, PasswordInput } from "../../../components/forms.js"

function SignInWithGoogle () {
	let rendered = false

	function renderButton (vnode) {
		if (window.google && !rendered) {
			google.accounts.id.renderButton(vnode.dom, {
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
				} else if (!res.body?.isSignedIn) {
					// Signing in with Google can trigger a sign up
					//
					// If that's the case, and the response was ok, then if they aren't
					// already signed in then it's likely to mean that their account
					// was created but requires manual activation by an admin through
					// user management, so we show them the sign up success page
					m.route.set(platform.routes.path("account.sign_up.success"))
				} else {
					platform.api.account.tryRedirect(platform.routes.path("home"))
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
				s.async = true
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

			return m("#sign-in__gsi_button.g_id_signin")
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
					console.error("the user either did not sign in to Facebook or did not authorise the app", response)

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
						} else if (!res.body?.isSignedIn) {
							// Signing in with Facebook can trigger a sign up
							//
							// If that's the case, and the response was ok, then if they aren't
							// already signed in then it's likely to mean that their account
							// was created but requires manual activation by an admin through
							// user management, so we show them the sign up success page
							m.route.set(platform.routes.path("account.sign_up.success"))
						} else {
							platform.api.account.tryRedirect(platform.routes.path("home"))
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
				s.async = true
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

			state.error = res.body?.error || ""
			state.errors = res.body?.fields || {}

			if (res.ok) {
				if (platform.session.isAwaitingTOTP) {
					m.route.set(platform.routes.path("account.sign_in.totp"))
				} else {
					platform.api.account.tryRedirect(platform.routes.path("home"))
				}
			} else if (res.status === platform.http.tooManyRequests) {
				let { inLast, unlockIn } = res.body

				if (unlockIn) {
					unlockIn = ` in ${unlockIn}`
				}

				state.error = `Too many failed sign in attempts in the last ${inLast}. Please try again${unlockIn}.`
			}
		})
	}

	return {
		view () {
			if (platform.session.isSignedIn) {
				return [
					m("p", "You're already signed in."),
					m("form", { onsubmit: signOut }, m(".bag", [
						m("button[type=submit].btn--link", "Click here to sign out."),
					])),
				]
			}

			return [
				m("h1", "Sign in"),
				m("form", { onsubmit: signIn }, [
					state.error ? m(ErrorBanner, state.error) : null,
					m(EmailInput, {
						label: "Email",
						name: "email",
						required: true,
						autocomplete: "username",
						error: state.errors.email,
						oninput (e) { state.email = e.target.value },
					}),
					m(PasswordInput, {
						label: "Password",
						name: "password",
						required: true,
						autocomplete: "current-password",
						error: state.errors.password,
						oninput (e) { state.password = e.target.value },
					}),
					m("button[type=submit]", "Sign in"),
					platform.config.signUpEnabled ? m(m.route.Link, { href: platform.routes.path("account.sign_up") }, "Sign up") : null,
					m(m.route.Link, { href: platform.routes.path("account.reset_password") }, "Forgotten your password?"),
					m("div", [
						m("p.sign-in-alt__title", "Or"),
						m(".sign-in-alt.text-center", [
							m(SignInWithGoogle),
							m(SignInWithFacebook),
						]),
					]),
				]),
			]
		},
	}
}

platform.routes.register("/account/sign-in", {
	name: "account.sign_in",
	render: SignIn,
})
