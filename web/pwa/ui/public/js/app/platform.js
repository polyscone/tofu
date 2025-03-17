import sdk from "{{.Asset.TagJSImport "/js/api/v1/sdk.js?type=module"}}"
import { show as showLoading, hide as hideLoading } from "{{.Asset.TagJSImport "./components/loading.js"}}"
import { open as openModal, close as closeModal } from "{{.Asset.TagJSImport "./components/modal.js"}}"

const langs = []

let lang = document.documentElement.getAttribute("lang")

if (lang === "en") {
	lang += "-GB"
}

if (lang) {
	langs.push(lang)
}

sdk.config = __STATE__.config || {}

sdk.routes = {
	__protected: [],
	__registered: {},
	protect (regexp) {
		sdk.routes.__protected.push(regexp)
	},
	register (pattern, opts) {
		opts ||= {}

		if (opts.name) {
			for (const pattern in sdk.routes.__registered) {
				const route = sdk.routes.__registered[pattern]

				if (route.name === opts.name) {
					throw new Error(`Duplicate route registered for the name "${opts.name}"`)
				}
			}
		}

		sdk.routes.__registered[pattern] = {
			name: opts.name || "",
			onmatch: opts.onmatch,
			render: opts.render,
		}
	},
	path (name, args) {
		for (const pattern in sdk.routes.__registered) {
			const route = sdk.routes.__registered[pattern]

			if (route.name === name) {
				const parts = pattern.split("/")

				for (let i = 0; i < parts.length; i++) {
					const part = parts[i]

					if (!part.startsWith(":")) {
						continue
					}

					const key = part.substring(1)

					if (key in args) {
						parts[i] = args[key]
					}
				}

				return parts.join("/")
			}
		}

		return `{${name}}`
	},
}

sdk.state = {
	loadingCount: 0,
	redirect: "",
}

sdk.tryRedirect = function tryRedirect (fallback) {
	if (sdk.state.redirect) {
		const redirect = sdk.state.redirect

		sdk.state.redirect = ""

		m.route.set(redirect)
	} else if (fallback) {
		m.route.set(fallback)
	}
},

sdk.loading = async function loading (p) {
	sdk.state.loadingCount++

	showLoading()

	try {
		if (typeof p === "function") {
			p = p()
		}

		return await p
	} catch (error) {
		console.error(error)
	} finally {
		sdk.state.loadingCount--

		if (sdk.state.loadingCount <= 0) {
			hideLoading()
		}
	}
}

sdk.modal = {
	open: openModal,
	close: closeModal,
}

sdk.localeString = function localeString (value, opts) {
	return value.toLocaleString(langs, opts)
}

sdk.localeDateString = function localeDateString (date, opts) {
	return date.toLocaleDateString(langs, opts)
}

window.platform = sdk

export default sdk
