{{define "master"}}{{template "master_js" .}}{{end}}

{{define "view_body" -}}

{{$factoryName := or (.URL.Query.Get "factory-name") "createSDK" -}}

function {{$factoryName | UnescapeJS}} (opts) {
	"use strict"

	opts ||= {}
	opts.host ||= ""

	if (!opts.fetch && typeof window !== "undefined") {
		opts.fetch = window.fetch
	}

	const host = opts.host
	const fetch = opts.fetch

	let _pollSessionHandle = null
	let _pollingSession = false
	let _pollNetworkStatusHandle = null
	let _pollingNetworkStatus = false
	let _isFirstRequest = true
	const _eventHandlers = {}

	async function _handle (event) {
		const listeners = _eventHandlers[event] || []

		for (const handler of listeners) {
			await handler()
		}
	}

	function addEventListener (event, func) {
		const valid = ["network", "session"]

		if (!valid.includes(event)) {
			throw new Error(`Event type must be one of: ["${valid.join('", "')}"]`)
		}

		if (typeof func !== "function") {
			throw new Error("Event handler must be a function")
		}

		_eventHandlers[event] ||= []

		_eventHandlers[event].push(func)
	}

	function removeEventListener (event, func) {
		const listeners = _eventHandlers[event] || []

		_eventHandlers[event] = listeners.filter(l => l !== func)
	}

	const account = {
		session () {
			if (typeof localStorage === "undefined") {
				return {}
			}

			const value = localStorage.getItem("sdk.session")

			if (!value) {
				return {}
			}

			return JSON.parse(value)
		},

		async pollSession () {
			if (typeof localStorage === "undefined") {
				return {}
			}

			_pollingSession = true

			clearTimeout(_pollSessionHandle)

			if (sdk.network === "connected") {
				const res = await request("/account/session")

				if (res.ok) {
					const prev = localStorage.getItem("sdk.session")
					const next = JSON.stringify(res.body)

					localStorage.setItem("sdk.session", next)

					if (prev !== next) {
						await _handle("session")
					}
				}
			}

			_pollSessionHandle = setTimeout(account.pollSession, 1 * 60 * 1000)
			_pollingSession = false
		},

		async signUp (email) {
			const res = await request("/account/sign-up", {
				method: "POST",
				body: { email },
			})

			return res
		},

		async verify (token, password, passwordCheck) {
			const res = await request("/account/verify", {
				method: "POST",
				body: { token, password, passwordCheck },
			})

			return res
		},

		async requestPasswordReset (email) {
			const res = await request("/account/reset-password", {
				method: "POST",
				body: { email },
			})

			return res
		},

		async resetPassword (token, newPassword, newPasswordCheck) {
			const res = await request("/account/reset-password/new-password", {
				method: "POST",
				body: { token, newPassword, newPasswordCheck },
			})

			return res
		},

		async signInWithPassword (email, password) {
			const res = await request("/account/sign-in", {
				method: "POST",
				body: { email, password },
			})

			await account.pollSession()

			return res
		},

		async requestSignInMagicLink (email) {
			const res = await request("/account/sign-in/magic-link/request", {
				method: "POST",
				body: { email },
			})

			return res
		},

		async signInWithMagicLink (token) {
			const res = await request("/account/sign-in/magic-link", {
				method: "POST",
				body: { token },
			})

			await account.pollSession()

			return res
		},

		async requestTOTPSMS () {
			const res = await request("/account/sign-in/totp/send-sms", {
				method: "POST",
			})

			return res
		},

		async signInWithTOTP (totp) {
			const res = await request("/account/sign-in/totp", {
				method: "POST",
				body: { totp },
			})

			await account.pollSession()

			return res
		},

		async signInWithRecoveryCode (recoveryCode) {
			const res = await request("/account/sign-in/recovery-code", {
				method: "POST",
				body: { recoveryCode },
			})

			await account.pollSession()

			return res
		},

		async signInWithGoogle (jwt) {
			const res = await request("/account/sign-in/google", {
				method: "POST",
				body: { jwt },
			})

			await account.pollSession()

			return res
		},

		async signInWithFacebook (userId, accessToken, email) {
			const res = await request("/account/sign-in/facebook", {
				method: "POST",
				body: { userId, accessToken, email },
			})

			await account.pollSession()

			return res
		},

		async signOut () {
			const res = await request("/account/sign-out", { method: "POST" })

			await account.pollSession()

			return res
		},
	}

	const system = {
		async config () {
			const res = await request("/system/config")

			return res
		},
	}

	const security = {
		csrfToken: null,

		async updateCSRFToken () {
			const res = await request("/security/csrf")

			if (res.ok) {
				security.csrfToken = res.body.csrfToken
			}
		},
	}

	const meta = {
		async pollNetworkStatus () {
			if (typeof navigator === "undefined") {
				return
			}

			_pollingNetworkStatus = true

			clearTimeout(_pollNetworkStatusHandle)

			const prev = sdk.network

			// navigator.onLine is only reliable when it's false
			// In this case we can just assume the network is offline and return
			if (!navigator.onLine) {
				sdk.network = "offline"
			} else {
				// If navigator.onLine is not false then we have to make an actual
				// request to determine the actual status of network connectivity
				const res = await request("/meta/health", { method: "HEAD" })

				sdk.network = res.ok ? "connected" : "disconnected"
			}

			if (sdk.network !== prev) {
				await _handle("network")
			}

			_pollNetworkStatusHandle = setTimeout(meta.pollNetworkStatus, 1 * 10 * 1000)
			_pollingNetworkStatus = false
		},
	}

	async function request (url, opts) {
		opts ||= {}
		opts.method ||= "GET"

		if (opts.body && typeof opts.body !== "string") {
			opts.body = JSON.stringify(opts.body)
		}

		if (opts.refreshCSRF || _isFirstRequest) {
			_isFirstRequest = false

			await security.updateCSRFToken()
		}

		if (!["GET", "HEAD", "OPTIONS", "TRACE"].includes(opts.method)) {
			opts.headers ||= {}
			opts.headers["x-csrf-token"] ||= security.csrfToken
		}

		const ret = {
			status: 0,
			ok: false,
			body: {},
			networkError: null,
		}

		const fullURL = host + sdk.prefix + url

		try {
			if (opts.body && (!opts.headers || !opts.headers["content-type"])) {
				opts.headers ||= {}
				opts.headers["content-type"] = "application/json"
			}

			const res = await fetch(fullURL, opts)

			security.csrfToken = res.headers.get("x-csrf-token") || security.csrfToken

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

			console.error(`api fetch failed: ${fullURL}: ${error}`)
		}

		ret.body ||= {}

		const maybeServerDown = ret.status === sdk.http.badGateway || ret.networkError

		if (maybeServerDown && !_pollingNetworkStatus) {
			await meta.pollNetworkStatus()
		}

		const maybeSessionExpired = [
			sdk.http.unauthorised,
			sdk.http.forbidden,
			sdk.http.internalServerError,
		].includes(ret.status)

		if (maybeSessionExpired && !_pollingSession) {
			await account.pollSession()
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
		addEventListener,
		removeEventListener,
		prefix: "{{.App.BasePath}}/api/v1",
		api: {
			account,
			system,
			security,
			meta,
		},
		request,
		get session () {
			return account.session()
		},
		http: {
			noContent: 204,
			badRequest: 400,
			unauthorised: 401,
			forbidden: 403,
			tooManyRequests: 429,
			internalServerError: 500,
			badGateway: 502,
			gatewayTimeout: 504,
		},
		network: "connected", // "offline" | "disconnected" | "connected",
	}

	return sdk
}

{{$globalName := .URL.Query.Get "global-name" -}}
{{if $globalName -}}
	{{printf "global[%q] ||= %v" $globalName $factoryName | UnescapeJS -}}
{{end -}}

{{if .URL.Query.Get "export" -}}
	export default {{$factoryName | UnescapeJS}}()
{{end -}}

{{- end}}
