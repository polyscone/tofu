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

							const onDestroyCallback = action.callback(node)

							if (typeof onDestroyCallback === "function") {
								onDestroy(node, onDestroyCallback)
							}
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

export function onMount (selector, callback) {
	_componentsInit()

	const nodes = Array.from(document.querySelectorAll(selector))

	for (const node of nodes) {
		const onDestroyCallback = callback(node)

		if (typeof onDestroyCallback === "function") {
			onDestroy(node, onDestroyCallback)
		}
	}

	window._components.mutationActions.mount.push({ selector, callback })
}

export function onDestroy (node, callback) {
	_componentsInit()

	window._components.mutationActions.destroy.push({ node, callback })
}
