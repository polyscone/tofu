onMount("textarea", node => {
	node.addEventListener("input", () => {
		node.style.height = "auto"
		node.style.height = node.scrollHeight + "px"
	})
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

onMount("[data-locale-number]", node => {
	node.innerHTML = node.innerHTML.replaceAll(/\d+(\.\d+)?/g, match => Number(match).toLocaleString([], {
		style: node.dataset.style,
		currency: node.dataset.currency,
		currencyDisplay: node.dataset.currencyDisplay || "narrowSymbol",
	}))
})

onMount("time", node => {
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
		node.innerText = date.toLocaleString([], {
			dateStyle: node.dataset.date || "short",
			timeStyle: node.dataset.time || "medium",
		})
	} else if (hasDate) {
		node.innerText = date.toLocaleDateString([], {
			dateStyle: node.dataset.date || "short",
		})
	} else if (hasTime) {
		node.innerText = date.toLocaleTimeString([], {
			timeStyle: node.dataset.time || "medium",
		})
	}
})

function _componentsInit () {
	window._components ||= {
		actions: {
			mount: [],
			destroy: [],
		},
		observer: new MutationObserver(mutations => {
			for (const mutation of mutations) {
				for (const node of mutation.addedNodes) {
					if (!node.matches) {
						continue
					}

					for (const action of window._components.actions.mount) {
						if (!node.matches(action.selector)) {
							continue
						}

						action.callback(node)
					}
				}

				for (const node of mutation.removedNodes) {
					if (!node.matches) {
						continue
					}

					for (const action of window._components.actions.destroy) {
						if (!node.matches(action.selector)) {
							continue
						}

						action.callback(node)
					}
				}
			}
		}),
	}

	window._components.observer.observe(document.body, {
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

	window._components.actions.mount.push({ selector, callback })
}

function onDestroy (selector, callback) {
	_componentsInit()

	window._components.actions.destroy.push({ selector, callback })
}
