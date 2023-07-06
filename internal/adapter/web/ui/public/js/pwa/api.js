var csrfToken = null

async function updateCSRFToken () {
	const res = await request("/api/v1/security/csrf")

	if (res.ok) {
		csrfToken = res.body.csrfToken
	}
}

async function request (url, opts) {
	opts ||= {}
	opts.method ||= "GET"

	if (opts.body && typeof opts.body !== "string") {
		opts.body = JSON.stringify(opts.body)
	}

	if (opts.refreshCSRF) {
		await updateCSRFToken()
	}

	if (!["GET", "HEAD", "OPTIONS", "TRACE"].includes(opts.method)) {
		opts.headers ||= {}
		opts.headers["x-csrf-token"] ||= csrfToken
	}

	const ret = {
		status: 0,
		ok: false,
		body: null,
		error: null,
	}

	try {
		const res = await fetch(url, opts)

		csrfToken = res.headers.get("x-csrf-token") || csrfToken

		ret.status = res.status
		ret.ok = res.ok

		if (res.status !== app.http.noContent) {
			if (res.headers.get("content-type") === "application/json") {
				ret.body = await res.json()
			} else {
				ret.body = await res.text()
			}
		}
	} catch (error) {
		ret.error = error

		console.error(`api fetch failed: ${error}: ${url}`)
	}

	if (!ret.ok && ret.body?.csrf && !opts.refreshCSRF) {
		opts.refreshCSRF = true

		return request(url, opts)
	}

	delete opts.refreshCSRF

	if (!ret.error && !ret.ok && ret.body?.fields) {
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

let pollSessionHandle = null

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

			if (app.isOnline) {
				const res = await request("/api/v1/account/session")

				if (res.ok) {
					localStorage.setItem("pwa.session", JSON.stringify(res.body))
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
		updateCSRFToken,
	},
	meta: {
		async updateOnlineStatus () {
			// We can't rely on navigator.onLine because when navigator.onLine is true
			// it doesn't necessarily mean we have a connection to the internet, we
			// could just be connected to any network
			//
			// We also want app.isOnline to reflect whether we can even contact
			// the server properly, so we need to ping an endpoint in the API anyway
			try {
				const res = await request("/api/v1/meta/health")

				// If res.ok is true then we got a valid response
				// Otherwise we assume we can't make a connection to the server
				app.isOnline = res.ok
			} catch {
				// A call to fetch will usually throw if we can't even send the request
				app.isOnline = false
			}
		},
	},
}

export default api
