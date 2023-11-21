import platform from "./platform.js"
import router from "./router.js"

window.addEventListener("load", async () => {
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
	const route = m.route.get() || "/"

	if (route !== __STATE__.url) {
		history.replaceState(null, "", "/")

		m.route.set(__STATE__.url)
	}
}

if ("serviceWorker" in navigator) {
	navigator.serviceWorker.register("/pwa_service_worker.js")
}
