function SignUpSuccess () {
	return {
		view: () => [
			m("p", "{{.T "pwa.account.sign_up.success.awaiting_activation"}}"),
			m("p", "{{.T "pwa.account.sign_up.success.once_activated"}}"),
			m(m.route.Link, { href: platform.routes.path("account.sign_in") }, "{{.T "pwa.account.sign_up.success.sign_in_link"}}"),
		],
	}
}

platform.routes.register("/account/sign-up/success", {
	name: "account.sign_up.success",
	render: SignUpSuccess,
})
