function SignUpSuccess () {
	return {
		view: () => [
			m("p", "Your account has been successfully verified and is now awaiting manual activation."),
			m("p", "Once your account is activated you'll receive an email and will be able to sign in."),
			m(m.route.Link, { href: platform.routes.path("account.sign_in") }, "Click here to go to the sign in page."),
		],
	}
}

platform.routes.register("/account/sign-up/success", {
	name: "account.sign_up.success",
	render: SignUpSuccess,
})
