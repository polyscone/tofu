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
			csrfToken = await getCSRFToken()
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

async function getCSRFToken () {
	const res = await request("/api/v1/security/csrf")

	return res.body.csrfToken
}

export default {
	auth: {
		async signInWithPassword (email, password) {
			return await request("/api/v1/account/sign-in", {
				method: "POST",
				body: { email, password },
			})
		},
		async signInWithTOTP (totp) {
			return await request("/api/v1/account/sign-in/totp", {
				method: "POST",
				body: { totp },
			})
		},
		async signInWithRecoveryCode (recoveryCode) {
			return await request("/api/v1/account/sign-in/recovery-code", {
				method: "POST",
				body: { recoveryCode },
			})
		},
		async signOut (email, password) {
			return await request("/api/v1/account/sign-out", { method: "POST" })
		},
	},
}
