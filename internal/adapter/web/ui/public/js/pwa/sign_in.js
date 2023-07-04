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

const SignInGoogle = {
	onupdate () {
		if (window.google) {
			const parent = document.getElementById("sign-in__gsi_button")

			if (parent) {
				google.accounts.id.renderButton(parent, {
					type: "standard",
					shape: "rectangle",
					theme: "outline",
					text: "signin_with",
					size: "large",
					logo_alignment: "center",
				})
			}
		}
	},
	view: () => !window.google ? null : [
		m("p.sign-in-alt__title", "Or"),
		m(".sign-in-alt.text-center", [
			m("#sign-in__gsi_button.g_id_signin"),
		]),
	],
}

const SignInPassword = {
	view: () => m("form", { onsubmit: signInWithPassword }, [
		state.error ? m(ErrorBanner, state.error) : null,
		m(EmailInput, {
			label: "Email",
			id: "sign-in__email",
			required: true,
			oninput (e) { state.email = e.target.value },
		}),
		m("p.error", state.errors.email),
		m(PasswordInput, {
			label: "Password",
			id: "sign-in__password",
			required: true,
			autocomplete: "current-password",
			oninput (e) { state.password = e.target.value },
		}),
		m("p.error", state.errors.password),
		m("button[type=submit]", "Sign in"),
		m(SignInGoogle),
	])
}

const SignInTOTP = {
	view: () => m("form", { onsubmit: signInWithTOTP }, [
		state.error ? m(ErrorBanner, state.error) : null,
		m(TOTPInput, {
			label: "Passcode",
			id: "sign-in__totp",
			required: true,
			autocomplete: "one-time-code",
			oninput (e) { state.totp = e.target.value },
		}),
		m("p.error", state.errors.totp),
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
			id: "sign-in__recovery-code",
			required: true,
			oninput (e) { state.recoveryCode = e.target.value },
		}),
		m("p.error", state.errors.recoveryCode),
		m(m.route.Link, { onclick: setScreen("totp") }, "Use a passcode"),
		m(m.route.Link, { onclick: setScreen("password") }, "Switch account"),
		m("button[type=submit]", "Sign in"),
	])
}

const SignInOffline = {
	view: () => m("p", "You must be online to sign in.")
}

const SignIn = {
	oncreate () {
		if (config.googleSignInClientId && !window.gsiLoaded) {
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
		let Component = null

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

		default:
			Component = SignInPassword
		}

		if (!app.online) {
			Component = SignInOffline
		}

		if (Component) {
			return m(".sign-in-splash", [
				m("h1", "Sign in"),
				m(Component),
			])
		}

		return null
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

		default:
			state.error = "Either this account does not exist, or your credentials are incorrect."
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
