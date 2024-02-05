{{define "master" -}}

let pollSessionHandle = null
let pollNetworkStatusHandle = null

const api = {
	account: {
		session () {
			const value = localStorage.getItem("sdk.session")

			if (!value) {
				return {}
			}

			return JSON.parse(value)
		},
		async pollSession () {
			clearTimeout(pollSessionHandle)

			if (sdk.network === "connected") {
				const res = await request(`${sdk.prefix}/account/session`)

				if (res.ok) {
					const prev = localStorage.getItem("sdk.session")
					const next = JSON.stringify(res.body)

					localStorage.setItem("sdk.session", next)

					if (prev !== next) {
						await sdk._handle("session")
					}
				}
			}

			pollSessionHandle = setTimeout(api.account.pollSession, 1 * 60 * 1000)
		},
		async signUp (email) {
			const res = await request(`${sdk.prefix}/account/sign-up`, {
				method: "POST",
				body: { email },
			})

			return res
		},
		async verify (token, password, passwordCheck) {
			const res = await request(`${sdk.prefix}/account/verify`, {
				method: "POST",
				body: { token, password, passwordCheck },
			})

			return res
		},
		async requestPasswordReset (email) {
			const res = await request(`${sdk.prefix}/account/reset-password`, {
				method: "POST",
				body: { email },
			})

			return res
		},
		async resetPassword (token, newPassword, newPasswordCheck) {
			const res = await request(`${sdk.prefix}/account/reset-password/new-password`, {
				method: "POST",
				body: { token, newPassword, newPasswordCheck },
			})

			return res
		},
		async signInWithPassword (email, password) {
			const res = await request(`${sdk.prefix}/account/sign-in`, {
				method: "POST",
				body: { email, password },
			})

			await api.account.pollSession()

			return res
		},
		async requestSignInMagicLink (email) {
			const res = await request(`${sdk.prefix}/account/sign-in/magic-link/request`, {
				method: "POST",
				body: { email },
			})

			return res
		},
		async signInWithMagicLink (token) {
			const res = await request(`${sdk.prefix}/account/sign-in/magic-link`, {
				method: "POST",
				body: { token },
			})

			await api.account.pollSession()

			return res
		},
		async requestTOTPSMS () {
			const res = await request(`${sdk.prefix}/account/sign-in/totp/send-sms`, {
				method: "POST",
			})

			return res
		},
		async signInWithTOTP (totp) {
			const res = await request(`${sdk.prefix}/account/sign-in/totp`, {
				method: "POST",
				body: { totp },
			})

			await api.account.pollSession()

			return res
		},
		async signInWithRecoveryCode (recoveryCode) {
			const res = await request(`${sdk.prefix}/account/sign-in/recovery-code`, {
				method: "POST",
				body: { recoveryCode },
			})

			await api.account.pollSession()

			return res
		},
		async signInWithGoogle (jwt) {
			const res = await request(`${sdk.prefix}/account/sign-in/google`, {
				method: "POST",
				body: { jwt },
			})

			await api.account.pollSession()

			return res
		},
		async signInWithFacebook (userId, accessToken, email) {
			const res = await request(`${sdk.prefix}/account/sign-in/facebook`, {
				method: "POST",
				body: { userId, accessToken, email },
			})

			await api.account.pollSession()

			return res
		},
		async signOut () {
			const res = await request(`${sdk.prefix}/account/sign-out`, { method: "POST" })

			await api.account.pollSession()

			return res
		},
	},
	system: {
		async config () {
			const res = await request(`${sdk.prefix}/system/config`)

			return res
		},
	},
	security: {
		csrfToken: null,
		async updateCSRFToken () {
			const res = await request(`${sdk.prefix}/security/csrf`)

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

			const prev = sdk.network

			// navigator.onLine is only reliable when it's false
			// In this case we can just assume the network is offline and return
			if (!navigator.onLine) {
				sdk.network = "offline"
			} else {
				// If navigator.onLine is not false then we have to make an actual
				// request to determine the actual status of network connectivity
				const res = await request(`${sdk.prefix}/meta/health`, { method: "HEAD" })

				sdk.network = res.ok ? "connected" : "disconnected"
			}

			if (sdk.network !== prev) {
				await sdk._handle("network")
			}

			pollNetworkStatusHandle = setTimeout(api.meta.pollNetworkStatus, 1 * 10 * 1000)
			api.meta.pollingNetworkStatus = false
		},
	},
}

let isFirstRequest = true

async function request (url, opts) {
	opts ||= {}
	opts.method ||= "GET"

	if (opts.body && typeof opts.body !== "string") {
		opts.body = JSON.stringify(opts.body)
	}

	if (opts.refreshCSRF || isFirstRequest) {
		isFirstRequest = false

		await api.security.updateCSRFToken()
	}

	if (!["GET", "HEAD", "OPTIONS", "TRACE"].includes(opts.method)) {
		opts.headers ||= {}
		opts.headers["x-csrf-token"] ||= api.security.csrfToken
	}

	const ret = {
		status: 0,
		ok: false,
		body: {},
		networkError: null,
	}

	try {
		const res = await fetch(url, opts)

		api.security.csrfToken = res.headers.get("x-csrf-token") || api.security.csrfToken

		ret.status = res.status
		ret.ok = res.ok

		if (res.status !== sdk.http.noContent) {
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

	ret.body ||= {}

	const maybeServerDown = ret.status === sdk.http.badGateway || ret.networkError

	if (maybeServerDown && !api.meta.pollingNetworkStatus) {
		await api.meta.pollNetworkStatus()
	}

	if (!ret.ok && ret.body.csrf && !opts.refreshCSRF) {
		opts.refreshCSRF = true

		return request(url, opts)
	}

	delete opts.refreshCSRF

	if (!ret.networkError && !ret.ok && ret.body.fields) {
		for (const key in ret.body.fields) {
			// Convert the key from space separated keys to camel case
			const newKey = key.replace(/\s+([a-z])/g, group => group.trim().toUpperCase())

			if (key !== newKey) {
				ret.body.fields[newKey] = ret.body.fields[key]

				delete ret.body.fields[key]
			}
		}
	}

	switch (ret.status) {
	case sdk.http.badGateway:
		ret.body.error = "Request failed because the server was offline, please try again."

		break

	case sdk.http.gatewayTimeout:
		ret.body.error = "The server could not process your request in time, please try again."

		break

	default:
		if (ret.networkError) {
			ret.body.error = "Request failed because either the server was offline, or you have no internet connection, please try again."
		}
	}

	return ret
}

const sdk = {
	_eventHandlers: {},
	async _handle (event) {
		const listeners = platform._eventHandlers[event] || []

		for (const handler of listeners) {
			await handler()
		}
	},
	addEventListener (event, func) {
		const valid = ["network", "session"]

		if (!valid.includes(event)) {
			throw new Error(`Event type must be one of: ["${valid.join('", "')}"]`)
		}

		if (typeof func !== "function") {
			throw new Error("Event handler must be a function")
		}

		sdk._eventHandlers[event] ||= []

		sdk._eventHandlers[event].push(func)
	},
	removeEventListener (event, func) {
		const listeners = platform._eventHandlers[event] || []

		platform._eventHandlers[event] = listeners.filter(l => l !== func)
	},
	prefix: "/api/v1",
	api,
	request,
	get session () {
		return api.account.session()
	},
	http: {
		noContent: 204,
		badRequest: 400,
		tooManyRequests: 429,
		badGateway: 502,
		gatewayTimeout: 504,
	},
	network: "connected", // "offline" | "disconnected" | "connected",
}

export default sdk

{{- end}}
