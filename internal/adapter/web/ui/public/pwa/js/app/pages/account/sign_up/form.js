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
				m.route.set(platform.routes.accountSignUpVerify.pattern)
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
				m("p", "You'll be able to choose your password after verifying your email address."),
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
					m(m.route.Link, { href: platform.routes.accountSignIn.pattern }, "Already have an account?"),
				]),
			]
		},
	}
}

platform.routes.accountSignUp = {
	pattern: "/account/sign-up",
	component: SignUp,
}
