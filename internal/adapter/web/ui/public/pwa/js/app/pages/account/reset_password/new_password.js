import { ErrorBanner } from "../../../components/error.js"
import { TokenInput, PasswordInput } from "../../../components/forms.js"

function ResetPasswordVerify () {
	const state = {
		error: "",
		errors: {},
		token: "",
		password: "",
	}

	function resetPassword (e) {
		e.preventDefault()

		platform.loading(async () => {
			const res = await platform.api.account.resetPassword(state.token, state.password, state.password)

			state.error = res.body?.error || ""
			state.errors = res.body?.fields || {}

			if (res.ok) {
				await platform.api.account.signInWithPassword(res.body?.email, state.password)

				platform.api.account.tryRedirect(platform.routes.home.pattern)
			}
		})
	}

	return {
		view: () => [
			m("p", "We've sent a verification code to your email address which you can copy into the input below."),
			m("form", { onsubmit: resetPassword }, [
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
				m("button[type=submit]", "Update password"),
			]),
		],
	}
}

platform.routes.accountResetPasswordVerify = {
	pattern: "/account/reset-password/new-password",
	component: ResetPasswordVerify,
}
