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

			state.error = res.body?.error || ""
			state.errors = res.body?.fields || {}

			if (res.ok) {
				platform.tryRedirect()
			}
		})
	}

	return {
		view: () => [
			m("h1", "Sign in"),
			m("form", { onsubmit: signIn }, [
				state.error ? m(ErrorBanner, state.error) : null,
				m(RecoveryCodeInput, {
					label: "Recovery code",
					name: "code",
					required: true,
					autocomplete: "off",
					error: state.errors.recoveryCode,
					oninput (e) { state.recoveryCode = e.target.value },
				}),
				m("button[type=submit]", "Sign in"),
				m(m.route.Link, { href: platform.routes.path("account.sign_in.totp") }, "Use a passcode"),
				m(m.route.Link, { href: platform.routes.path("account.sign_in") }, "Switch account"),
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
