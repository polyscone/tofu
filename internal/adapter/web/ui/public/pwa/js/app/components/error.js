export function ErrorBanner () {
	return {
		view: vnode => m(".error-banner", m("p", vnode.children))
	}
}
