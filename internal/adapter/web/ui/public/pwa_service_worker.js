const cacheName = "pwa.assets"
const strategies = {
	networkOnly: [
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
			"/all/css/common.css",
			"/pwa/css/pwa.css",
			"/pwa/js/mithril/v2.2.2.min.js",
			"/pwa/js/app/main.js",
		])
	}

	event.waitUntil(handle())
})

self.addEventListener("activate", event => {})

self.addEventListener("fetch", event => {
	const url = new URL(event.request.url)

	async function handle () {
		if (strategies.networkOnly.includes(url.pathname) || event.request.method !== "GET") {
			try {
				return await fetch(event.request)
			} catch (error) {
				console.error(`service worker fetch failed: ${error}: ${event.request.url}`)

				return new Response("", { status: 500, statusText: "Offline" })
			}
		}

		try {
			const fetched = await fetch(event.request)
			const clone = fetched.clone()

			caches.open(cacheName).then(cache => {
				cache.put(event.request, clone)
			})

			return fetched
		} catch (error) {
			const cached = await caches.match(event.request)

			if (cached) {
				return cached
			}

			console.error(`service worker fetch failed: ${error}: ${event.request.url}`)

			return new Response("", { status: 500, statusText: "Offline" })
		}
	}

	event.respondWith(handle())
})
