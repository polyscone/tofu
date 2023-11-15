const SignOut = {
	view () {
		if (!platform.session.isSignedIn) {
			return null
		}

		return m("form", { onsubmit: signOut }, [
			m("button[type=submit].btn--link", "Sign out"),
		])
	}
}

async function signOut (e) {
	e.preventDefault()

	platform.loading.show()

	await platform.api.account.signOut()

	platform.loading.hide()
}

export default SignOut
