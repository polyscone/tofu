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
			ret.body =  await res.text()
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
		async signIn (email, password) {
			return await request("/api/v1/account/sign-in", {
				method: "POST",
				body: { email, password },
			})
		},
		async signOut (email, password) {
			return await request("/api/v1/account/sign-out", { method: "POST" })
		},
	},
}
