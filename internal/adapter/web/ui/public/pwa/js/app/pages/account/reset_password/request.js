import { ErrorBanner } from "../../../components/error.js"
import { EmailInput } from "../../../components/forms.js"

function ResetPassword () {
	const state = {
		error: "",
		errors: {},
		email: "",
	}

	function requestPasswordReset (e) {
		e.preventDefault()

		platform.loading(async () => {
			const res = await platform.api.account.requestPasswordReset(state.email)

			state.error = res.body?.error || ""
			state.errors = res.body?.fields || {}

			if (res.ok) {
				m.route.set(platform.routes.accountResetPasswordVerify.pattern)
			}
		})
	}

	return {
		view: () => [
			m("form", { onsubmit: requestPasswordReset }, [
				state.error ? m(ErrorBanner, state.error) : null,
				m(EmailInput, {
					label: "Email",
					name: "email",
					required: true,
					autocomplete: "username",
					error: state.errors.email,
					oninput (e) { state.email = e.target.value },
				}),
				m("button[type=submit]", "Send reset password code"),
			]),
		],
	}
}

platform.routes.accountResetPassword = {
	pattern: "/account/reset-password",
	component: ResetPassword,
}
