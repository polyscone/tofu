const SignOut = {
	view () {
		if (!app.session.isSignedIn) {
			return null
		}

		return m("form", { onsubmit: signOut }, [
			m("button[type=submit].btn--link", "Sign out"),
		])
	}
}

async function signOut (e) {
	e.preventDefault()

	app.loading.show()

	await app.api.account.signOut()

	app.loading.hide()
}

export default SignOut
