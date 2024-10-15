import platform from "./platform.js"
import router from "./router.js"

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
	navigator.serviceWorker.register("{{.App.BasePath}}/pwa_service_worker.js")
}

onMount("textarea", node => {
	node.addEventListener("input", () => {
		node.style.height = "auto"
		node.style.height = node.scrollHeight + "px"
	})
})

function _componentsInit () {
	window._components ||= {
		actions: {
			mount: [],
			destroy: [],
		},
		observer: new MutationObserver(mutations => {
			for (const mutation of mutations) {
				for (const node of mutation.addedNodes) {
					if (!node.matches) {
						continue
					}

					for (const action of window._components.actions.mount) {
						if (!node.matches(action.selector)) {
							continue
						}

						action.callback(node)
					}
				}

				for (const node of mutation.removedNodes) {
					if (!node.matches) {
						continue
					}

					for (const action of window._components.actions.destroy) {
						if (!node.matches(action.selector)) {
							continue
						}

						action.callback(node)
					}
				}
			}
		}),
	}

	window._components.observer.observe(document.body, {
		childList: true,
		subtree: true,
	})
}

function onMount (selector, callback) {
	_componentsInit()

	const nodes = Array.from(document.querySelectorAll(selector))

	for (const node of nodes) {
		callback(node)
	}

	window._components.actions.mount.push({ selector, callback })
}

function onDestroy (selector, callback) {
	_componentsInit()

	window._components.actions.destroy.push({ selector, callback })
}
