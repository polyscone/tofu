import { ErrorBanner } from "../../../components/error.js"
import { RecoveryCodeInput } from "../../../components/forms.js"

function SignInRecoveryCode () {
	const state = {
		error: "",
		errors: {},
		recoveryCode: "",
	}

	function signIn (e) {
		e.preventDefault()

		platform.loading(async () => {
			const res = await platform.api.account.signInWithRecoveryCode(state.recoveryCode)

			state.error = res.body.error || ""
			state.errors = res.body.fields || {}

			if (res.ok) {
				platform.tryRedirect()
			}
		})
	}

	return {
		view: () => [
			m("h1", "{{.T "pwa.account.sign_in.recovery_code.title"}}"),
			m("form", { onsubmit: signIn }, [
				state.error ? m(ErrorBanner, state.error) : null,
				m(RecoveryCodeInput, {
					label: "{{.T "pwa.account.sign_in.recovery_code.recovery_code_label"}}",
					name: "code",
					required: true,
					autocomplete: "off",
					error: state.errors.recoveryCode,
					oninput (e) { state.recoveryCode = e.target.value },
				}),
				m("button[type=submit]", "{{.T "pwa.account.sign_in.recovery_code.sign_in_button"}}"),
				m(m.route.Link, { href: platform.routes.path("account.sign_in.totp") }, "{{.T "pwa.account.sign_in.recovery_code.use_passcode_button"}}"),
				m(m.route.Link, { href: platform.routes.path("account.sign_in") }, "{{.T "pwa.account.sign_in.recovery_code.switch_account_button"}}"),
			]),
		],
	}
}

platform.routes.register("/account/sign-in/recovery-code", {
	name: "account.sign_in.recovery_code",
	onmatch () {
		if (platform.session.isSignedIn || !platform.session.isAwaitingTOTP) {
			return m.route.set(platform.routes.path("account.sign_in"))
		}
	},
	render: SignInRecoveryCode,
})
