import { onMount, onDestroy } from "{{.Asset.TagImport "/site/js/components.js"}}"

const langs = []

let lang = document.documentElement.getAttribute("lang")

if (lang === "en") {
	lang += "-GB"
}

if (lang) {
	langs.push(lang)
}

onMount("textarea", node => {
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

	window._components.observeResize(node, () => {
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
		console.log("On destroy running...")

		window.removeEventListener("blur", unsetMouseDown)
	}
})

onMount("input, textarea", node => {
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

onMount("time", node => {
	if (typeof node.dataset.skip !== "undefined") {
		return
	}

	let str = (node.getAttribute("datetime") || node.innerText).trim()

	if (!str.trim()) {
		return
	}

	const hasDate = /\d{4}-\d{2}-\d{2}/.test(str)
	const hasTime = /\d{2}:\d{2}/.test(str)

	if (!hasDate && hasTime) {
		const now = (new Date()).toISOString().split("T").shift()

		str = now + "T" + str
	}

	const date = new Date(str)

	if (hasDate && hasTime) {
		node.innerText = date.toLocaleString(langs, {
			dateStyle: node.dataset.date || "short",
			timeStyle: node.dataset.time || "medium",
		})
	} else if (hasDate) {
		node.innerText = date.toLocaleDateString(langs, {
			dateStyle: node.dataset.date || "short",
		})
	} else if (hasTime) {
		node.innerText = date.toLocaleTimeString(langs, {
			timeStyle: node.dataset.time || "medium",
		})
	}
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
