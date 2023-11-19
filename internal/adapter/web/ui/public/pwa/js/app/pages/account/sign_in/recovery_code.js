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
				platform.api.account.tryRedirect()
			}
		})
	}

	return {
		view: () => m("form", { onsubmit: signIn }, [
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
			m(m.route.Link, { href: platform.routes.accountSignInTOTP.pattern }, "Use a passcode"),
			m(m.route.Link, { href: platform.routes.accountSignIn.pattern }, "Switch account"),
		]),
	}
}

platform.routes.accountSignInRecoveryCode = {
	pattern: "/account/sign-in/recovery-code",
	component: SignInRecoveryCode,
	onmatch () {
		if (platform.session.isSignedIn || !platform.session.isAwaitingTOTP) {
			return m.route.set(platform.routes.accountSignIn.pattern)
		}
	},
}
