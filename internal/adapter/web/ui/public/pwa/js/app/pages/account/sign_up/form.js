import { ErrorBanner } from "../../../components/error.js"
import { EmailInput } from "../../../components/forms.js"

function SignUp () {
	const state = {
		error: "",
		errors: {},
		email: "",
	}

	function signOut (e) {
		e.preventDefault()

		platform.loading(platform.api.account.signOut)
	}

	function signUp (e) {
		e.preventDefault()

		platform.loading(async () => {
			const res = await platform.api.account.signUp(state.email)

			state.error = res.body?.error || ""
			state.errors = res.body?.fields || {}

			if (res.ok) {
				m.route.set(platform.routes.path("account.sign_up.verify"))
			}
		})
	}

	return {
		view () {
			if (platform.session.isSignedIn) {
				return [
					m("p", "You're already signed in."),
					m("form", { onsubmit: signOut }, [
						m("button[type=submit].btn--link", "Click here to sign out."),
					]),
				]
			}

			return [
				m("h1", "Sign up"),
				m("form", { onsubmit: signUp }, [
					state.error ? m(ErrorBanner, state.error) : null,
					m(EmailInput, {
						label: "Email",
						name: "email",
						required: true,
						autocomplete: "username",
						error: state.errors.email,
						oninput (e) { state.email = e.target.value },
					}),
					m("button[type=submit]", "Sign up"),
					m(m.route.Link, { href: platform.routes.path("account.sign_in") }, "Already have an account?"),
				]),
			]
		},
	}
}

platform.routes.register("/account/sign-up", {
	name: "account.sign_up",
	render: SignUp,
})
