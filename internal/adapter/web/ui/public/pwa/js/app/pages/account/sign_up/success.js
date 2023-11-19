function SignUpSuccess () {
	return {
		view: () => [
			m("p", "Your account has been successfully verified and is now awaiting manual activation."),
			m("p", "Once your account is activated you'll receive an email and will be able to sign in."),
			m(m.route.Link, { href: platform.routes.accountSignIn.pattern }, "Click here to go to the sign in page."),
		],
	}
}

platform.routes.accountSignUpSuccess = {
	pattern: "/account/sign-up/success",
	component: SignUpSuccess,
}
