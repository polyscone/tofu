function SignInTOTPRequired () {
	return {
		view: () => m("p", "{{.T "pwa.account.sign_in.totp_required.text"}}"),
	}
}

platform.routes.register("/account/sign-in/totp/required", {
	name: "account.sign_in.totp_required",
	onmatch () {
		if (!platform.session.isSignedIn) {
			return m.route.set(platform.routes.path("account.sign_in"))
		}

		if (!platform.session.isTOTPRequired) {
			return m.route.set(platform.routes.path("home"))
		}
	},
	render: SignInTOTPRequired,
})
