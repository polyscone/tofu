import SignIn from "./sign_in.js"
import SignOut from "./sign_out.js"
import Loading from "./loading.js"

const Layout = {
	view: vnode => [
		m("header.header", [
			m("h1", "PWA"),
			m(SignOut),
		]),
		m("main.main", vnode.children),
		m(SignIn),
		m(Loading),
	],
}

export default Layout
