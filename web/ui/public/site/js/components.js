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

	onDestroy(node, () => {
		window.removeEventListener("blur", windowblur)
	})

	window._components.observeResize(node, () => {
		userResized ||= mouseDown
	})

	function autosize () {
		if (userResized) {
			return
		}

		const computed = getComputedStyle(node)
		const borderBlock = parseInt(computed.borderWidth) * 2
		const height = node.scrollHeight + borderBlock + "px"

		if (height != node.style.height) {
			node.style.height = "auto"
			node.style.height = height
		}
	}

	autosize()

	node.addEventListener("input", autosize)
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

function _flattenNodes (node) {
	const nodes = [node]

	for (const child of node.childNodes) {
		nodes.push(child)

		const flattened = _flattenNodes(child)

		for (const grandchild of flattened) {
			nodes.push(grandchild)
		}
	}

	return nodes
}

function _flattenNodesUnique (node) {
	return [...new Set(_flattenNodes(node))]
}

function _componentsInit () {
	window._components ||= {
		initialised: false,
		resizeActions: [],
		resizeObserver: new ResizeObserver(entries => {
			for (const entry of entries) {
				for (const action of window._components.resizeActions) {
					if (entry.target !== action.node) {
						continue
					}

					action.callback(entry)
				}
			}
		}),
		observeResize (node, callback) {
			window._components.resizeActions.push({ node, callback })
			window._components.resizeObserver.observe(node)
		},
		mutationActions: {
			mount: [],
			destroy: [],
		},
		mutationObserver: new MutationObserver(mutations => {
			const seenAdded = []
			const seenRemoved = []

			for (const mutation of mutations) {
				for (const node of mutation.addedNodes) {
					const nodes = _flattenNodesUnique(node)

					for (const node of nodes) {
						if (seenAdded.includes(node) || !node.matches) {
							continue
						}

						seenAdded.push(node)

						for (const action of window._components.mutationActions.mount) {
							if (!node.matches(action.selector)) {
								continue
							}

							action.callback(node)
						}
					}
				}

				for (const node of mutation.removedNodes) {
					const nodes = _flattenNodesUnique(node)

					for (const node of nodes) {
						if (seenRemoved.includes(node) || !node.matches) {
							continue
						}

						seenRemoved.push(node)

						for (const action of window._components.mutationActions.destroy) {
							if (node !== action.node) {
								continue
							}

							action.callback(node)
						}
					}
				}
			}
		}),
	}

	if (window._components.initialised) {
		return
	}

	window._components.initialised = true

	window._components.mutationObserver.observe(document.body, {
		childList: true,
		subtree: true,
	})
}

function onMount (selector, callback) {
	_componentsInit()

	const nodes = Array.from(document.querySelectorAll(selector))

	for (const node of nodes) {
		callback(node)
	}

	window._components.mutationActions.mount.push({ selector, callback })
}

function onDestroy (node, callback) {
	_componentsInit()

	window._components.mutationActions.destroy.push({ node, callback })
}

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
