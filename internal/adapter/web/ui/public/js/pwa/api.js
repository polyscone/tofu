var csrfToken = null

async function request (url, opts) {
	opts ||= {}
	opts.method ||= "GET"

	if (opts.body && typeof opts.body !== "string") {
		opts.body = JSON.stringify(opts.body)
	}

	if (opts.refreshCSRF) {
		const res = await request("/api/v1/security/csrf")

		if (res.ok) {
			csrfToken = res.body.csrfToken
		}
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

		console.error(`api fetch failed: ${error}`)
	}

	if (!ret.ok && ret.body?.csrf && !opts.refreshCSRF) {
		opts.refreshCSRF = true

		return request(url, opts)
	}

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

let updateSessionHandle = null

const api = {
	account: {
		session () {
			const value = localStorage.getItem("pwa.session")

			if (!value) {
				return {}
			}

			return JSON.parse(value)
		},
		async updateSession () {
			clearTimeout(updateSessionHandle)

			if (app.online) {
				const res = await request("/api/v1/account/session")

				if (res.ok) {
					localStorage.setItem("pwa.session", JSON.stringify(res.body))
				}
			}

			updateSessionHandle = setTimeout(api.account.updateSession, 1 * 60 * 1000)
		},
		async signInWithPassword (email, password) {
			const res = await request("/api/v1/account/sign-in", {
				method: "POST",
				body: { email, password },
			})

			await api.account.updateSession()

			return res
		},
		async signInWithTOTP (totp) {
			const res = await request("/api/v1/account/sign-in/totp", {
				method: "POST",
				body: { totp },
			})

			await api.account.updateSession()

			return res
		},
		async signInWithRecoveryCode (recoveryCode) {
			const res = await request("/api/v1/account/sign-in/recovery-code", {
				method: "POST",
				body: { recoveryCode },
			})

			await api.account.updateSession()

			return res
		},
		async signOut (email, password) {
			const res = await request("/api/v1/account/sign-out", { method: "POST" })

			await api.account.updateSession()

			return res
		},
	},
}

export default api
