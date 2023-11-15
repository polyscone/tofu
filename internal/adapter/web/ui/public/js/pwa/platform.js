import { show, hide } from "./loading.js"

let pollSessionHandle = null
let pollNetworkStatusHandle = null

const api = {
	account: {
		session () {
			const value = localStorage.getItem("pwa.session")

			if (!value) {
				return {}
			}

			return JSON.parse(value)
		},
		async pollSession () {
			clearTimeout(pollSessionHandle)

			if (platform.network === "connected") {
				const res = await request("/api/v1/account/session")

				if (res.ok) {
					const prev = localStorage.getItem("pwa.session")
					const next = JSON.stringify(res.body)

					localStorage.setItem("pwa.session", next)

					if (prev !== next) {
						m.redraw()
					}
				}
			}

			pollSessionHandle = setTimeout(api.account.pollSession, 1 * 60 * 1000)
		},
		async signInWithPassword (email, password) {
			const res = await request("/api/v1/account/sign-in", {
				method: "POST",
				body: { email, password },
			})

			await api.account.pollSession()

			return res
		},
		async signInWithTOTP (totp) {
			const res = await request("/api/v1/account/sign-in/totp", {
				method: "POST",
				body: { totp },
			})

			await api.account.pollSession()

			return res
		},
		async signInWithRecoveryCode (recoveryCode) {
			const res = await request("/api/v1/account/sign-in/recovery-code", {
				method: "POST",
				body: { recoveryCode },
			})

			await api.account.pollSession()

			return res
		},
		async signInWithGoogle (jwt) {
			const res = await request("/api/v1/account/sign-in/google", {
				method: "POST",
				body: { jwt },
			})

			await api.account.pollSession()

			return res
		},
		async signOut (email, password) {
			const res = await request("/api/v1/account/sign-out", { method: "POST" })

			await api.account.pollSession()

			return res
		},
	},
	security: {
		csrfToken: null,
		async updateCSRFToken () {
			const res = await request("/api/v1/security/csrf")

			if (res.ok) {
				api.security.csrfToken = res.body.csrfToken
			}
		},
	},
	meta: {
		pollingNetworkStatus: false,
		async pollNetworkStatus () {
			api.meta.pollingNetworkStatus = true

			clearTimeout(pollNetworkStatusHandle)

			const prev = platform.network

			// navigator.onLine is only reliable when it's false
			// In this case we can just assume the network is offline and return
			if (!navigator.onLine) {
				platform.network = "offline"
			} else {
				// If navigator.onLine is not false then we have to make an actual
				// request to determine the actual status of network connectivity
				const res = await request("/api/v1/meta/health", { method: "HEAD" })

				platform.network = res.ok ? "connected" : "disconnected"
			}

			if (platform.network !== prev) {
				m.redraw()
			}

			pollNetworkStatusHandle = setTimeout(api.meta.pollNetworkStatus, 1 * 10 * 1000)
			api.meta.pollingNetworkStatus = false
		},
	},
}

const platform = {
	api,
	config: __STATE__.config,
	get session () {
		return api.account.session()
	},
	loading: {
		show,
		hide,
	},
	http: {
		noContent: 204,
		tooManyRequests: 429,
		badGateway: 502,
	},
	network: "connected", // "offline" | "disconnected" | "connected"
}

export default platform

async function request (url, opts) {
	opts ||= {}
	opts.method ||= "GET"

	if (opts.body && typeof opts.body !== "string") {
		opts.body = JSON.stringify(opts.body)
	}

	if (opts.refreshCSRF) {
		await api.security.updateCSRFToken()
	}

	if (!["GET", "HEAD", "OPTIONS", "TRACE"].includes(opts.method)) {
		opts.headers ||= {}
		opts.headers["x-csrf-token"] ||= api.security.csrfToken
	}

	const ret = {
		status: 0,
		ok: false,
		body: null,
		networkError: null,
	}

	try {
		const res = await fetch(url, opts)

		api.security.csrfToken = res.headers.get("x-csrf-token") || api.security.csrfToken

		ret.status = res.status
		ret.ok = res.ok

		if (res.status !== platform.http.noContent) {
			if (res.headers.get("content-type") === "application/json") {
				ret.body = await res.json()
			} else {
				ret.body = await res.text()
			}
		}
	} catch (error) {
		ret.networkError = error

		console.error(`api fetch failed: ${error}: ${url}`)
	}

	const maybeServerDown = ret.status === platform.http.badGateway || ret.networkError

	if (maybeServerDown && !api.meta.pollingNetworkStatus) {
		await api.meta.pollNetworkStatus()
	}

	if (!ret.ok && ret.body?.csrf && !opts.refreshCSRF) {
		opts.refreshCSRF = true

		return request(url, opts)
	}

	delete opts.refreshCSRF

	if (!ret.networkError && !ret.ok && ret.body?.fields) {
		for (const key in ret.body.fields) {
			// Convert the key from space separated keys to camel case
			const newKey = key.replace(/\s+([a-z])/g, group => group.trim().toUpperCase())

			if (key !== newKey) {
				ret.body.fields[newKey] = ret.body.fields[key]

				delete ret.body.fields[key]
			}
		}
	}

	return ret
}
