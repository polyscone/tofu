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

const SignIn = {
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
	state.errors = res.body.fields || {}

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
	state.errors = res.body.fields || {}

	app.loading.hide()
}

async function signInWithRecoveryCode (e) {
	e.preventDefault()

	app.loading.show()

	const res = await app.api.account.signInWithRecoveryCode(state.recoveryCode)

	state.error = ""
	state.errors = res.body.fields || {}

	app.loading.hide()
}

function setScreen (screen) {
	return e => {
		e.preventDefault()

		state.error = ""
		state.errors = {}
		state.screen = screen
	}
}

export default SignIn
