import { ErrorBanner } from "../../../components/error.js"
import { TokenInput, PasswordInput } from "../../../components/forms.js"

function SignUpVerify () {
	const state = {
		error: "",
		errors: {},
		token: "",
		password: "",
	}

	function signOut (e) {
		e.preventDefault()

		platform.loading(platform.api.account.signOut)
	}

	function verify (e) {
		e.preventDefault()

		platform.loading(async () => {
			const res = await platform.api.account.verify(state.token, state.password, state.password)

			// Save any errors for display later
			const error = res.body?.error || ""
			const errors = res.body?.fields || {}

			if (res.ok) {
				if (res.body?.isActivated) {
					await platform.api.account.signInWithPassword(res.body?.email, state.password)

					platform.api.account.tryRedirect(platform.routes.home.pattern)
				} else {
					m.route.set(platform.routes.accountSignUpSuccess.pattern)
				}
			} else {
				// If we couldn't verify the sign up then it's possible that the user
				// was given a password reset token instead because they were already signed up
				const res = await platform.api.account.resetPassword(state.token, state.password, state.password)

				if (res.ok) {
					await platform.api.account.signInWithPassword(res.body?.email, state.password)

					platform.api.account.tryRedirect(platform.routes.home.pattern)
				} else {
					// Finally display any original errors
					// Since this is a sign up verification page, and reset password is only
					// secondary functionality, we don't want to show password reset errors
					state.error = error
					state.errors = errors
				}
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
				m("form", { onsubmit: verify }, [
					state.error ? m(ErrorBanner, state.error) : null,
					m(TokenInput, {
						label: "Verification code",
						name: "token",
						required: true,
						error: state.errors.token,
						oninput (e) { state.token = e.target.value },
					}),
					m(PasswordInput, {
						label: "Choose a password",
						name: "password",
						required: true,
						autocomplete: "new-password",
						error: state.errors.password,
						oninput (e) { state.password = e.target.value },
					}),
					m("button[type=submit]", "Use this password and verify account"),
				]),
			]
		},
	}
}

platform.routes.accountSignUpVerify = {
	pattern: "/account/sign-up/verify",
	component: SignUpVerify,
}
