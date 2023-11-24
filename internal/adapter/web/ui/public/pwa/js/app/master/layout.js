import Loading from "../components/loading.js"
import Modal from "../components/modal.js"

function SignOut () {
	function signOut (e) {
		e.preventDefault()

		platform.loading(async () => {
			await platform.api.account.signOut()

			// Run the router again to do any automatic redirect logic etc.
			m.route.set(m.route.get())
		})
	}

	return {
		view () {
			if (!platform.session.isSignedIn) {
				return null
			}

			return m("form", { onsubmit: signOut }, [
				m("button[type=submit].btn--link", "Sign out"),
			])
		},
	}
}

function Layout () {
	return {
		view: vnode => [
			m("header.header", [
				m("h1", "PWA"),
				m(SignOut),
			]),
			m("main.main", vnode.children),
			m(Modal),
			m(Loading),
		],
	}
}

export default Layout
