import { ErrorBanner } from "./error.js"
import { EmailInput, PasswordInput, TOTPInput, RecoveryCodeInput } from "./forms.js"

const state = {
	error: "",
	errors: {},
	email: "",
	password: "",
	totp: "",
	recoveryCode: "",
}

const SignOut = {
	view: () => [
		m("p", "You are already signed in."),
		m(m.route.Link, { href: "?sign-out", onclick: signOut }, "Click here to sign out."),
	],
}

export const SignInPassword = {
	view: () => app.auth.isSignedIn ? m(SignOut) : [
		m("form", { onsubmit: signInWithPassword }, [
			state.error ? m(ErrorBanner, state.error) : null,
			m(EmailInput, {
				label: "Email",
				id: "sign-in__email",
				required: true,
				value: state.email,
				oninput (e) { state.email = e.target.value },
			}),
			m("p.error", state.errors.email),
			m(PasswordInput, {
				label: "Password",
				id: "sign-in__password",
				required: true,
				autocomplete: "current-password",
				value: state.password,
				oninput (e) { state.password = e.target.value },
			}),
			m("p.error", state.errors.password),
			m("button[type=submit]", "Sign in"),
		])
	]
}

export const SignInTOTP = {
	view: () => app.auth.isSignedIn ? m(SignOut) : [
		m("form", { onsubmit: signInWithTOTP }, [
			state.error ? m(ErrorBanner, state.error) : null,
			m(TOTPInput, {
				label: "Passcode",
				id: "sign-in__totp",
				required: true,
				autocomplete: "one-time-code",
				value: state.totp,
				oninput (e) { state.totp = e.target.value },
			}),
			m("p.error", state.errors.totp),
			m(m.route.Link, { href: "/sign-in/recovery-code" }, "Use a recovery code"),
			m("button[type=submit]", "Sign in"),
		])
	]
}

export const SignInRecoveryCode = {
	view: () => app.auth.isSignedIn ? m(SignOut) : [
		m("form", { onsubmit: signInWithRecoveryCode }, [
			state.error ? m(ErrorBanner, state.error) : null,
			m(RecoveryCodeInput, {
				label: "Recovery code",
				id: "sign-in__recovery-code",
				required: true,
				value: state.recoveryCode,
				oninput (e) { state.recoveryCode = e.target.value },
			}),
			m("p.error", state.errors.recoveryCode),
			m(m.route.Link, { href: "/sign-in/totp" }, "Use a passcode"),
			m("button[type=submit]", "Sign in"),
		])
	]
}

async function signOut (e) {
	e.preventDefault()

	app.loading.show()

	const res = await app.api.auth.signOut()

	state.error = ""
	state.errors = {}

	if (res.ok) {
		app.auth.isSignedIn = false

		m.route.set("/sign-in")
	}

	app.loading.hide()
}

async function signInWithPassword (e) {
	e.preventDefault()

	app.loading.show()

	const res = await app.api.auth.signInWithPassword(state.email, state.password)

	state.error = ""
	state.errors = res.body.fields || {}

	if (res.ok) {
		app.auth.isSignedIn = res.body.isSignedIn
		app.auth.isAwaitingTOTP = res.body.isAwaitingTOTP

		let next = app.auth.next || "/"

		if (app.auth.isAwaitingTOTP) {
			next = "/sign-in/totp"
		}

		m.route.set(next)
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

	const res = await app.api.auth.signInWithTOTP(state.totp)

	state.error = ""
	state.errors = res.body.fields || {}

	if (res.ok) {
		app.auth.isSignedIn = res.body.isSignedIn
		app.auth.isAwaitingTOTP = false

		m.route.set(app.auth.next || "/")
	}

	app.loading.hide()
}

async function signInWithRecoveryCode (e) {
	e.preventDefault()

	app.loading.show()

	const res = await app.api.auth.signInWithRecoveryCode(state.recoveryCode)

	state.error = ""
	state.errors = res.body.fields || {}

	if (res.ok) {
		app.auth.isSignedIn = res.body.isSignedIn
		app.auth.isAwaitingTOTP = false

		m.route.set(app.auth.next || "/")
	}

	app.loading.hide()
}
