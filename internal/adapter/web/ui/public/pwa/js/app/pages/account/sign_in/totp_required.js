function SignInTOTPRequired () {
	return {
		view: () => m("p", "Two-factor authentication is required to use this application."),
	}
}

platform.routes.accountSignInTOTPRequired = {
	pattern: "/account/sign-in/totp/required",
	component: SignInTOTPRequired,
	onmatch () {
		if (!platform.session.isSignedIn) {
			return m.route.set(platform.routes.accountSignIn.pattern)
		}

		if (!platform.session.isTOTPRequired) {
			return m.route.set(platform.routes.home.pattern)
		}
	},
}
