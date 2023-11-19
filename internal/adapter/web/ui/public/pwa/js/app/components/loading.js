const state = {
	visible: false,
	handle: null,
}

export function show () {
	clearTimeout(state.handle)

	state.handle = setTimeout(() => {
		state.visible = true

		m.redraw()
	}, 200)
}

export function hide () {
	clearTimeout(state.handle)

	state.visible = false

	m.redraw()
}

export function Loading () {
	return {
		view: () => m(".loading", { class: state.visible ? "loading--show" : null }, [
			m(".loading__spinner"),
		]),
	}
}

export default Loading
