import { ErrorBanner } from "./error.js"
import { EmailInput, PasswordInput } from "./forms.js"

const state = {
	error: '',
	email: '',
	password: '',
}

const SignIn = {
	view: () => app.auth.isSignedIn ? [
		m("p", "You are already signed in."),
		m("a", { href: "?sign-out", onclick: signOut }, "Click here to sign out."),
	] : [
		m("form", { onsubmit: signIn }, [
			state.error ? m(ErrorBanner, state.error) : null,
			m(EmailInput, {
				label: "Email",
				id: "sign-in__email",
				required: true,
				value: state.email,
				oninput (e) { state.email = e.target.value },
			}),
			m(PasswordInput, {
				label: "Password",
				id: "sign-in__password",
				required: true,
				autocomplete: "current-password",
				value: state.password,
				oninput (e) { state.password = e.target.value },
			}),
			m("button[type=submit]", "Sign in"),
		])
	]
}

async function signIn (e) {
	e.preventDefault()

	app.loading.show()

	const res = await app.api.auth.signIn(state.email, state.password)

	if (res.ok) {
		app.auth.isSignedIn = true

		if (!app.auth.redirect) {
			app.auth.redirect = "/"
		}

		m.route.set(app.auth.redirect)
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

async function signOut (e) {
	e.preventDefault()

	app.loading.show()

	const res = await app.api.auth.signOut()

	if (res.ok) {
		app.auth.isSignedIn = false
	}

	app.loading.hide()
}

export default SignIn
