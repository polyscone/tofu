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

			state.error = res.body.error || ""
			state.errors = res.body.fields || {}

			if (res.ok) {
				await platform.api.account.signInWithPassword(res.body.email, state.password)

				platform.tryRedirect(platform.routes.path("home"))
			}
		})
	}

	return {
		view: () => [
			m("h1", "{{.T "pwa.account.reset_password.new_password.title"}}"),
			m("p", "{{.T "pwa.account.reset_password.new_password.sent_verification_code_text"}}"),
			m("form", { onsubmit: resetPassword }, [
				state.error ? m(ErrorBanner, state.error) : null,
				m(TokenInput, {
					label: "{{.T "pwa.account.reset_password.new_password.verification_code_label"}}",
					name: "token",
					required: true,
					error: state.errors.token,
					oninput (e) { state.token = e.target.value },
				}),
				m(PasswordInput, {
					label: "{{.T "pwa.account.reset_password.new_password.choose_password_label"}}",
					name: "password",
					required: true,
					autocomplete: "new-password",
					error: state.errors.password,
					oninput (e) { state.password = e.target.value },
				}),
				m("button[type=submit]", "{{.T "pwa.account.reset_password.new_password.update_password_button"}}"),
			]),
		],
	}
}

platform.routes.register("/account/reset-password/new-password", {
	name: "account.reset_password.verify",
	render: ResetPasswordVerify,
})
