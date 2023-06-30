var csrfToken = null

async function request (url, opts) {
	opts ||= {}

	if (!opts.method) {
		opts.method = "GET"
	}

	if (opts.body) {
		opts.body = JSON.stringify(opts.body)
	}

	if (!["GET", "HEAD", "OPTIONS", "TRACE"].includes(opts.method)) {
		if (!csrfToken) {
			const res = await request("/api/v1/security/csrf")

			if (res.ok) {
				csrfToken = res.body.csrfToken
			}
		}

		opts.headers ||= {}

		if (!opts.headers["x-csrf-token"]) {
			opts.headers["x-csrf-token"] = csrfToken
		}
	}

	const res = await fetch(url, opts)

	csrfToken = res.headers.get("x-csrf-token") || csrfToken

	const ret = {
		status: res.status,
		ok: res.ok,
		body: null,
	}

	if (res.status !== app.http.noContent) {
		if (res.headers.get("content-type") === "application/json") {
			ret.body = await res.json()
		} else {
			ret.body = await res.text()
		}
	}

	if (!ret.ok && ret.body?.fields) {
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

			if (navigator.onLine) {
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
