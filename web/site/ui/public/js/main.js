import { onMount, observeResize } from "{{.Asset.TagJSImport "./dom.js"}}"

const langs = []

let lang = document.documentElement.getAttribute("lang")

if (lang === "en") {
	lang += "-GB"
}

if (lang) {
	langs.push(lang)
}

window.sdk ||= {}
window.sdk.browser = {
	form: {
		async enable (form, p) {
			try {
				if (typeof p === "function") {
					p = p()
				}

				return await p
			} catch (error) {
				console.error(error)
			} finally {
				if (!form) {
					return
				}

				delete form.dataset.submitting

				const disabled = form.querySelectorAll("[data-disable]")

				for (const node of disabled) {
					if (node.dataset.originalTextContent) {
						node.textContent = node.dataset.originalTextContent

						delete node.dataset.originalTextContent
					}

					node.classList.remove("btn--loading")

					delete node.dataset.disable
				}
			}
		}
	}
}

window.addEventListener("pageshow", () => {
	onMount("[data-submitting]", node => {
		delete node.dataset.submitting
	})

	onMount("[data-disable]", node => {
		if (node.dataset.originalTextContent) {
			node.textContent = node.dataset.originalTextContent

			delete node.dataset.originalTextContent
		}

		node.classList.remove("btn--loading")

		delete node.dataset.disable
	})
})

onMount("form", node => {
	node.addEventListener("submit", event => {
		if (event.target.dataset.submitting) {
			event.preventDefault()

			return
		}

		if (event.target.method.toLowerCase() === "get") {
			return
		}

		event.target.dataset.submitting = true

		if (!event.submitter.classList.contains("btn--link")) {
			event.submitter.classList.add("btn--loading")
			event.submitter.dataset.originalTextContent = event.submitter.textContent
			event.submitter.textContent = "{{.T "site.form.please_wait_button_text"}}"
		}

		const buttons = node.querySelectorAll(":is(button, .btn, input[type=button], input[type=submit]):not(.btn--link)")

		for (const button of buttons) {
			// We don't want to actually disable any buttons here just in case
			// they're being used to send a value to the server, so we set
			// a data attribute instead
			button.dataset.disable = "loading"
		}
	})
})

onMount("textarea[data-autosize]", node => {
	let mouseDown = false
	let userResized = false

	node.addEventListener("mousedown", () => {
		mouseDown = true
	})

	function unsetMouseDown () {
		mouseDown = false
	}

	node.addEventListener("mouseup", unsetMouseDown)
	window.addEventListener("blur", unsetMouseDown)

	observeResize(node, () => {
		userResized ||= mouseDown
	})

	function autosize () {
		// If the user explicitly resized the textarea themselves then
		// we assume they want to keep it that size and don't do anything
		if (userResized) {
			return
		}

		node.style.height = "auto"

		const hasScrollbar = node.clientHeight < node.scrollHeight
		const nodeHeight = hasScrollbar ? node.scrollHeight : node.clientHeight
		const computed = getComputedStyle(node)
		const additionalBlock = (parseInt(computed.paddingBlock) + parseInt(computed.borderWidth)) * 2
		const height = nodeHeight + additionalBlock + "px"

		if (height != node.style.height) {
			node.style.height = height
		}
	}

	autosize()

	node.addEventListener("input", autosize)

	return () => {
		window.removeEventListener("blur", unsetMouseDown)
	}
})

onMount("form input, form textarea", node => {
	// Set a data-invalid attribute on forms when they're submitted with
	// malformed inputs
	//
	// This is to allow for styling invalid form elements after submittal
	// in a more persistent way than is allowed with CSS only
	node.addEventListener("invalid", () => {
		const form = node.closest("form")

		if (form) {
			form.dataset.invalid = true
		}
	})
})

onMount(".invalid", node => {
	node.addEventListener("input", () => {
		node.classList.remove("invalid")
	})
})

onMount("[data-char-count-for]", node => {
	const id = node.dataset.charCountFor
	const el = document.getElementById(id)

	if (!el) {
		return
	}

	const max = Number(el.getAttribute("maxlength"))

	if (max <= 0 || isNaN(max)) {
		return
	}

	node.innerText = formatNumber(el.value.length) + "/" + formatNumber(max)

	el.addEventListener("input", () => {
		node.innerText = formatNumber(el.value.length) + "/" + formatNumber(max)
	})
})

onMount("[data-locale-number]", node => {
	node.innerHTML = node.innerHTML.replaceAll(/\d+(\.\d+)?/g, match => formatNumber(match, node.dataset))
})

onMount("[data-locale-date], [data-locale-time]", node => {
	const dateStyle = node.dataset.localeDate === "" ? "short" : node.dataset.localeDate
	const timeStyle = node.dataset.localeTime === "" ? "medium" : node.dataset.localeTime

	if (typeof dateStyle === "undefined" && typeof timeStyle === "undefined") {
		return
	}

	let str = (node.getAttribute("datetime") || node.innerText).trim()

	if (!str) {
		return
	}

	const hasDate = /\d{4}-\d{2}-\d{2}/.test(str)
	const hasTime = /\d{2}:\d{2}/.test(str)

	if (!hasDate && !hasTime) {
		return
	}

	if (!hasDate && hasTime) {
		const now = (new Date()).toISOString().split("T").shift()

		str = now + "T" + str
	}

	const date = new Date(str)

	node.innerText = date.toLocaleString(langs, { dateStyle, timeStyle })
})

function formatNumber (number, opts) {
	number = Number(number)

	if (isNaN(number)) {
		return
	}

	opts ||= {}

	return number.toLocaleString(langs, {
		style: opts.style,
		currency: opts.currency,
		currencyDisplay: opts.currencyDisplay || "narrowSymbol",
	})
}
