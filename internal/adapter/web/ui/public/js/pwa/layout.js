import Loading from "./loading.js"

const Layout = {
	view: vnode => [
		m("header", [
			m("h1", `${app.name} PWA`),
		]),
		vnode.children,
		m(Loading),
	],
}

export default Layout
