const state = {
	title: "Title",
	content: "Content",
	actions: [
		{
			text: "Cancel",
			classes: "",
			onclick () {
				platform.modal.close()
			},
		},
	],
	visible: false,
}

export function open (title, content, actions) {
	if (!Array.isArray(actions)) {
		actions = []
	}

	state.title = String(title)
	state.content = String(content)
	state.actions = actions

	state.visible = true

	m.redraw()
}

export function close () {
	state.visible = false

	m.redraw()
}

export function Modal () {
	return {
		view () {
			let classes = ""

			if (state.visible) {
				classes = "modal--open"
			}

			const actions = []

			for (const action of state.actions) {
				actions.push(m("button", {
					class: action.classes,
					onclick: action.onclick,
				}, action.text))
			}

			return m(".modal", { class: classes }, [
				m(".modal__inner", [
					m(".modal__title", state.title),
					m(".modal__content", state.content),
					m(".modal__actions", actions),
				]),
			])
		},
	}
}

export default Modal
