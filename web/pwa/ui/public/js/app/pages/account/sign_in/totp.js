import { ErrorBanner } from "../../../components/error.js"
import { TOTPInput } from "../../../components/forms.js"

function wait () {
	return new Promise(resolve => {
		setTimeout(resolve, 3000)
	})
}

function SignInTOTP () {
	const state = {
		error: "",
		errors: {},
		totp: "",
		resendText: "{{.T "pwa.account.sign_in.totp.resend_passcode_button"}}",
		resendingSMS: false,
	}

	function resendTOTPSMS (e) {
		e.preventDefault()

		// Don't allow more messages to be sent until the current request is finished
		if (state.resendingSMS) {
			return
		}

		state.resendingSMS = true

		const originalText = state.resendText

		state.resendText = "{{.T "pwa.account.sign_in.totp.resend_passcode_button_please_wait"}}"

		platform.loading(async () => {
			const res = await platform.api.account.requestTOTPSMS(state.totp)

			state.error = res.body.error || ""
			state.errors = res.body.fields || {}

			if (res.ok) {
				state.resendText = "{{.T "pwa.account.sign_in.totp.resend_passcode_button_sent"}}"

				// Give the user some time to see the message in the button change
				// and prevent them from sending multiple messages at once
				setTimeout(() => {
					state.resendText = originalText
					state.resendingSMS = false

					m.redraw()
				}, 3000)
			} else {
				state.resendText = originalText
			}
		})
	}

	function signIn (e) {
		e.preventDefault()

		platform.loading(async () => {
			const res = await platform.api.account.signInWithTOTP(state.totp)

			state.error = res.body.error || ""
			state.errors = res.body.fields || {}

			if (res.ok) {
				platform.tryRedirect(platform.routes.path("home"))
			}
		})
	}

	return {
		view () {
			let instructions = "{{.T "app:pwa.account.sign_in.totp.instruction_text"}}"
			let smsButton = null

			if (platform.session.totpMethod === "sms") {
				instructions = "{{.T "sms:pwa.account.sign_in.totp.instruction_text"}}"
				smsButton = m("button.btn--alt", { onclick: resendTOTPSMS }, state.resendText)
			}

			return [
				m("h1", "{{.T "pwa.account.sign_in.totp.title"}}"),
				m("p", instructions),
				m("form", { onsubmit: signIn }, [
					state.error ? m(ErrorBanner, state.error) : null,
					m(TOTPInput, {
						label: "{{.T "pwa.account.sign_in.totp.passcode_label"}}",
						name: "totp",
						required: true,
						autocomplete: "one-time-code",
						error: state.errors.totp,
						oninput (e) { state.totp = e.target.value },
					}),
					m("button[type=submit]", "{{.T "pwa.account.sign_in.totp.sign_in_button"}}"),
					smsButton,
					m(m.route.Link, { href: platform.routes.path("account.sign_in.recovery_code") }, "{{.T "pwa.account.sign_in.totp.use_recovery_code_button"}}"),
					m(m.route.Link, { href: platform.routes.path("account.sign_in") }, "{{.T "pwa.account.sign_in.totp.switch_account_button"}}"),
				]),
			]
		},
	}
}

platform.routes.register("/account/sign-in/totp", {
	name: "account.sign_in.totp",
	onmatch () {
		if (platform.session.isSignedIn || !platform.session.isAwaitingTOTP) {
			return m.route.set(platform.routes.path("account.sign_in"))
		}
	},
	render: SignInTOTP,
})
