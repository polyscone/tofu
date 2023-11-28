const state = {
	visible: false,
	handle: null,
}

export function show () {
	clearTimeout(state.handle)

	state.handle = setTimeout(() => {
		state.visible = true

		m.redraw()
	}, 300)
}

export function hide () {
	clearTimeout(state.handle)

	state.visible = false

	m.redraw()
}

export function Loading () {
	return {
		view () {
			let classes = ""

			if (state.visible) {
				classes = "loading--show"
			}

			return m(".loading", { class: classes }, [
				m(".loading__spinner"),
			])
		},
	}
}

export default Loading
