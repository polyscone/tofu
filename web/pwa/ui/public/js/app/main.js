import { onMount, observeResize } from "{{.Asset.TagJSImport "/js/dom.js"}}"
import platform from "{{.Asset.TagJSImport "./platform.js"}}"
import router from "{{.Asset.TagJSImport "./router.js"}}"

window.addEventListener("load", async () => {
	platform.addEventListener("network", () => {
		// Refresh the current page allowing route logic to run again
		m.route.set(m.route.get())
	})

	platform.addEventListener("session", () => {
		// Refresh the current page allowing route logic to run again
		m.route.set(m.route.get())
	})

	await platform.api.meta.pollNetworkStatus()
	await platform.api.account.pollSession()
	await platform.api.security.updateCSRFToken()

	m.redraw()
})

window.addEventListener("online", platform.api.meta.pollNetworkStatus)
window.addEventListener("offline", platform.api.meta.pollNetworkStatus)

router()

// If the Mithril router is configured to use a prefix then we
// want to make sure that any non-prefixed requests replace the
// path in the URL with / and instead set the router state to
// whatever the URL was, but this time with the prefix
if (platform.config.prefix) {
	const route = m.route.get() || "{{.App.BasePath}}/"

	if (route !== __STATE__.url) {
		history.replaceState(null, "", "{{.App.BasePath}}/")

		m.route.set(__STATE__.url)
	}
}

if ("serviceWorker" in navigator) {
	navigator.serviceWorker.register("{{.App.BasePath}}/service_worker.js", { type: "module" })
}

onMount("textarea[data-autosize]", node => {
	let mouseDown = false
	let userResized = false

	node.addEventListener("mousedown", () => {
		mouseDown = true
	})

	function unsetMouseDown () {
		mouseDown = false
	}

	node.addEventListener("mouseup", unsetMouseDown)
	window.addEventListener("blur", unsetMouseDown)

	observeResize(node, () => {
		userResized ||= mouseDown
	})

	function autosize () {
		// If the user explicitly resized the textarea themselves then
		// we assume they want to keep it that size and don't do anything
		if (userResized) {
			return
		}

		node.style.height = "auto"

		const hasScrollbar = node.clientHeight < node.scrollHeight
		const nodeHeight = hasScrollbar ? node.scrollHeight : node.clientHeight
		const computed = getComputedStyle(node)
		const additionalBlock = (parseInt(computed.paddingBlock) + parseInt(computed.borderWidth)) * 2
		const height = nodeHeight + additionalBlock + "px"

		if (height != node.style.height) {
			node.style.height = height
		}
	}

	autosize()

	node.addEventListener("input", autosize)

	return () => {
		window.removeEventListener("blur", unsetMouseDown)
	}
})
