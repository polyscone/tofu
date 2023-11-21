import { ErrorBanner } from "../../../components/error.js"
import { EmailInput, PasswordInput } from "../../../components/forms.js"

function SignInWithGoogle () {
	let rendered = false

	function renderButton (vnode) {
		if (window.google && !rendered) {
			const parent = vnode.dom.querySelector("#sign-in__gsi_button")

			google.accounts.id.renderButton(parent, {
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
					m.route.set(platform.routes.accountSignInTOTP.pattern)
				} else if (!res.body?.isSignedIn) {
					// Signing in with Google can trigger a sign up
					//
					// If that's the case, and the response was ok, then if they aren't
					// already signed in then it's likely to mean that their account
					// was created but requires manual activation by an admin through
					// user management, so we show them the sign up success page
					m.route.set(platform.routes.accountSignUpSuccess.pattern)
				} else {
					platform.api.account.tryRedirect(platform.routes.home.pattern)
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

			return m("div", [
				m("p.sign-in-alt__title", "Or"),
				m(".sign-in-alt.text-center", [
					m("#sign-in__gsi_button.g_id_signin"),
				]),
			])
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
					m.route.set(platform.routes.accountSignInTOTP.pattern)
				} else {
					platform.api.account.tryRedirect(platform.routes.home.pattern)
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

			return m("form", { onsubmit: signIn }, [
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
				platform.config.signUpEnabled ? m(m.route.Link, { href: platform.routes.accountSignUp.pattern }, "Sign up") : null,
				m(m.route.Link, { href: platform.routes.accountResetPassword.pattern }, "Forgotten your password?"),
				m(SignInWithGoogle),
			])
		},
	}
}

platform.routes.register("/account/sign-in", SignIn, {
	name: "account.sign_in",
})
