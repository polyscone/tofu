const cacheName = "pwa.assets"
const strats = {
	network: [
		"/api/v1/account/session",
		"/api/v1/security/csrf",
	],
}

self.addEventListener("install", event => {
	async function handle () {
		const cache = await caches.open(cacheName)

		return cache.addAll([
			"/",
			"/pwa_service_worker.js",
			"/css/common.css",
			"/css/pwa.css",
			"/js/mithril/v2.2.2.min.js",
			"/js/pwa/app.js",
		])
	}

	event.waitUntil(handle())
})

self.addEventListener("activate", event => {})

self.addEventListener("fetch", event => {
	const url = new URL(event.request.url)

	async function handle () {
		if (strats.network.includes(url.pathname) || event.request.method !== "GET") {
			try {
				return await fetch(event.request)
			} catch (error) {
				console.error(`service worker fetch failed: ${error}: ${event.request.url}`)

				return new Response("", { status: 500, statusText: "Offline" })
			}
		}

		const cached = await caches.match(event.request)
		const fetched = fetch(event.request).then(res => {
			const clone = res.clone()

			caches.open(cacheName).then(cache => {
				cache.put(event.request, clone)
			})

			return res
		}).catch(error => {
			console.error(`service worker fetch failed: ${error}: ${event.request.url}`)

			return new Response("", { status: 500, statusText: "Offline" })
		})

		return cached || fetched
	}

	event.respondWith(handle())
})
