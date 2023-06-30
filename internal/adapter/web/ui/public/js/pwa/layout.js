import SignIn from "./sign_in.js"
import Loading from "./loading.js"

const Layout = {
	view: vnode => [
		m("header.header", [
			m("h1", "PWA"),
		]),
		m("main.main", vnode.children),
		!app.session.isSignedIn ? m(SignIn) : null,
		m(Loading),
	],
}

export default Layout
