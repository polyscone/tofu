const components = {
	resizeActions: [],
	resizeObserver: new ResizeObserver(entries => {
		for (const entry of entries) {
			for (const action of components.resizeActions) {
				if (entry.target !== action.node) {
					continue
				}

				action.callback(entry)
			}
		}
	}),
	mutationActions: {
		mount: [],
		unmount: [],
	},
	mutationObserver: new MutationObserver(mutations => {
		const seenAdded = []
		const seenRemoved = []

		for (const mutation of mutations) {
			for (const node of mutation.addedNodes) {
				const nodes = flattenNodesUnique(node)

				for (const node of nodes) {
					if (seenAdded.includes(node) || !node.matches) {
						continue
					}

					seenAdded.push(node)

					for (const action of components.mutationActions.mount) {
						if (!node.matches(action.selector)) {
							continue
						}

						const onUnmountCallback = action.callback(node)

						if (typeof onUnmountCallback === "function") {
							onUnmount(node, onUnmountCallback)
						}
					}
				}
			}

			for (const node of mutation.removedNodes) {
				const nodes = flattenNodesUnique(node)

				for (const node of nodes) {
					if (seenRemoved.includes(node) || !node.matches) {
						continue
					}

					seenRemoved.push(node)

					for (const action of components.mutationActions.unmount) {
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

components.mutationObserver.observe(document.body, {
	childList: true,
	subtree: true,
})

function flattenNodes (node) {
	const nodes = [node]

	for (const child of node.childNodes) {
		nodes.push(child)

		const flattened = flattenNodes(child)

		for (const grandchild of flattened) {
			nodes.push(grandchild)
		}
	}

	return nodes
}

function flattenNodesUnique (node) {
	return [...new Set(flattenNodes(node))]
}

export function onMount (selector, callback) {
	const nodes = Array.from(document.querySelectorAll(selector))

	for (const node of nodes) {
		const onUnmountCallback = callback(node)

		if (typeof onUnmountCallback === "function") {
			onUnmount(node, onUnmountCallback)
		}
	}

	components.mutationActions.mount.push({ selector, callback })
}

export function onUnmount (node, callback) {
	components.mutationActions.unmount.push({ node, callback })
}

export function observeResize (node, callback) {
	components.resizeActions.push({ node, callback })
	components.resizeObserver.observe(node)
}
