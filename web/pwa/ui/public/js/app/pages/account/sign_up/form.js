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

			state.error = res.body.error || ""
			state.errors = res.body.fields || {}

			if (res.ok) {
				m.route.set(platform.routes.path("account.sign_up.verify"))
			}
		})
	}

	return {
		view () {
			if (platform.session.isSignedIn) {
				return [
					m("p", "{{.T "pwa.account.sign_up.already_signed_in.text"}}"),
					m("form", { onsubmit: signOut }, [
						m("button[type=submit].btn--link", "{{.T "pwa.account.sign_up.already_signed_in.sign_out_button"}}"),
					]),
				]
			}

			return [
				m("h1", "{{.T "pwa.account.sign_up.form.title"}}"),
				m("form", { onsubmit: signUp }, [
					state.error ? m(ErrorBanner, state.error) : null,
					m(EmailInput, {
						label: "{{.T "pwa.account.sign_up.form.email_label"}}",
						name: "email",
						required: true,
						autocomplete: "username",
						error: state.errors.email,
						oninput (e) { state.email = e.target.value },
					}),
					m("button[type=submit]", "{{.T "pwa.account.sign_up.form.sign_up_button"}}"),
					m(m.route.Link, { href: platform.routes.path("account.sign_in") }, "{{.T "pwa.account.sign_up.form.sign_in_button"}}"),
				]),
			]
		},
	}
}

platform.routes.register("/account/sign-up", {
	name: "account.sign_up",
	render: SignUp,
})
