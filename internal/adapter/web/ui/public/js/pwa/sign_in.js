import { ErrorBanner } from "./error.js"
import { EmailInput, PasswordInput, TOTPInput, RecoveryCodeInput } from "./forms.js"

const state = {
	screen: "password",
	error: "",
	errors: {},
	email: "",
	password: "",
	totp: "",
	recoveryCode: "",
}

function SignInGoogle () {
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

	return {
		oncreate: renderButton,
		onupdate: renderButton,
		onremove () {
			rendered = false
		},
		view: () => (config.googleSignInEnabled && window.google) ? m("div", [
			m("p.sign-in-alt__title", "Or"),
			m(".sign-in-alt.text-center", [
				m("#sign-in__gsi_button.g_id_signin"),
			]),
		]) : null,
	}
}

const SignInPassword = {
	view: () => m("form", { onsubmit: signInWithPassword }, [
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
		m(SignInGoogle),
	])
}

const SignInTOTP = {
	view: () => m("form", { onsubmit: signInWithTOTP }, [
		state.error ? m(ErrorBanner, state.error) : null,
		m(TOTPInput, {
			label: "Passcode",
			name: "totp",
			required: true,
			autocomplete: "one-time-code",
			error: state.errors.totp,
			oninput (e) { state.totp = e.target.value },
		}),
		m(m.route.Link, { onclick: setScreen("recoveryCode") }, "Use a recovery code"),
		m(m.route.Link, { onclick: setScreen("password") }, "Switch account"),
		m("button[type=submit]", "Sign in"),
	])
}

const SignInRecoveryCode = {
	view: () => m("form", { onsubmit: signInWithRecoveryCode }, [
		state.error ? m(ErrorBanner, state.error) : null,
		m(RecoveryCodeInput, {
			label: "Recovery code",
			name: "code",
			required: true,
			autocomplete: "off",
			error: state.errors.recoveryCode,
			oninput (e) { state.recoveryCode = e.target.value },
		}),
		m(m.route.Link, { onclick: setScreen("totp") }, "Use a passcode"),
		m(m.route.Link, { onclick: setScreen("password") }, "Switch account"),
		m("button[type=submit]", "Sign in"),
	])
}

const SignInTOTPRequired = {
	view: () => m("p", "Two-factor authentication on sign in is required to use this application.")
}

const SignInOffline = {
	view: () => m("p", "You must be online to sign in.")
}

const SignInDisconnected = {
	view: () => m("p", "Either the server is offline, or you have no internet connection, please try again later.")
}

const SignIn = {
	oncreate () {
		if (config.googleSignInEnabled && config.googleSignInClientId && !window.gsiLoaded) {
			window.gsiLoaded = true

			const s = document.createElement("script")

			s.src = "https://accounts.google.com/gsi/client"
			s.async = true
			s.defer = true
			s.onload = function () {
				google.accounts.id.initialize({
					client_id: config.googleSignInClientId,
					context: "signin",
					ux_mode: "popup",
					async callback (res) {
						await signInWithGoogle(res.credential)

						m.redraw()
					},
				})

				m.redraw()
			}

			document.body.appendChild(s)
		}
	},
	view () {
		const showSignIn = !app.session.isSignedIn || app.session.isTOTPRequired && !app.session.totpMethod

		if (!showSignIn) {
			return null
		}

		let Component = SignInPassword

		switch (state.screen) {
		case "password":
			Component = SignInPassword

			break

		case "totp":
			Component = SignInTOTP

			break

		case "recoveryCode":
			Component = SignInRecoveryCode

			break

		case "totpRequired":
			Component = SignInTOTPRequired

			break
		}

		switch (app.network) {
		case "offline":
			Component = SignInOffline

			break

		case "disconnected":
			Component = SignInDisconnected

			break
		}

		return m(".sign-in-splash", [
			m("h1", "Sign in"),
			m(Component),
		])
	}
}

async function signInWithPassword (e) {
	e.preventDefault()

	app.loading.show()

	const res = await app.api.account.signInWithPassword(state.email, state.password)

	state.error = ""
	state.errors = res.body?.fields || {}

	if (res.ok) {
		if (app.session.isAwaitingTOTP) {
			state.screen = "totp"
		} else if (app.session.isTOTPRequired) {
			state.screen = "totpRequired"
		}
	} else {
		switch (res.status) {
		case app.http.tooManyRequests:
			let { inLast, unlockIn } = res.body

			if (unlockIn) {
				unlockIn = ` in ${unlockIn}`
			}

			state.error = `Too many failed sign in attempts in the last ${inLast}. Please try again${unlockIn}.`

			break

		case app.http.badGateway:
			state.error = "Sign in failed because the server was offline, please try again."

			break

		default:
			if (res.networkError) {
				state.error = "Sign in failed because either the server was offline, or you have no internet connection, please try again."
			} else {
				state.error = "Your credentials are incorrect."
			}
		}
	}

	app.loading.hide()
}

async function signInWithTOTP (e) {
	e.preventDefault()

	app.loading.show()

	const res = await app.api.account.signInWithTOTP(state.totp)

	state.error = ""
	state.errors = res.body?.fields || {}

	app.loading.hide()
}

async function signInWithRecoveryCode (e) {
	e.preventDefault()

	app.loading.show()

	const res = await app.api.account.signInWithRecoveryCode(state.recoveryCode)

	state.error = ""
	state.errors = res.body?.fields || {}

	app.loading.hide()
}

async function signInWithGoogle (jwt) {
	app.loading.show()

	const res = await app.api.account.signInWithGoogle(jwt)

	state.error = ""
	state.errors = res.body?.fields || {}

	if (!res.ok) {
		switch (res.status) {
		case app.http.badGateway:
			state.error = "Sign in failed because the server was offline, please try again."

			break

		default:
			if (res.networkError) {
				state.error = "Sign in failed because either the server was offline, or you have no internet connection, please try again."
			} else {
				state.error = "Your credentials are incorrect."
			}
		}
	}

	app.loading.hide()
}

function setScreen (screen) {
	return event => {
		event.preventDefault()

		state.error = ""
		state.errors = {}
		state.screen = screen
	}
}

export default SignIn
