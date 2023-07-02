const cacheName = "pwa.assets"

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
	async function handle () {
		const cached = await caches.match(event.request)
		const fetched = fetch(event.request).then(res => {
			const clone = res.clone()

			caches.open(cacheName).then(cache => {
				cache.put(event.request, clone)
			})

			return res
		}).catch(error => {
			console.error(`service worker fetch failed: ${error}`)
		})

		return cached || fetched
	}

	event.respondWith(handle())
})
