export const EmailInput = {
	view: (vnode) => [
		vnode.attrs.label ? m("label", {
			for: vnode.attrs.id,
		}, vnode.attrs.label) : null,
		m("input[type=email]", {
			oninput: vnode.attrs.oninput,
			value: vnode.attrs.value,
			id: vnode.attrs.id,
			required: vnode.attrs.required,
			placeholder: "email@example.com",
			autocomplete: "email",
			pattern: "^[a-zA-Z0-9+._~\\-]+@[a-zA-Z0-9+._~\\-]+(\.[a-zA-Z0-9+._~\\-]+)+$",
			maxlength: 100,
		}),
	]
}

export const PasswordInput = {
	view: (vnode) => [
		vnode.attrs.label ? m("label", {
			for: vnode.attrs.id,
		}, vnode.attrs.label) : null,
		m("input[type=password]", {
			oninput: vnode.attrs.oninput,
			value: vnode.attrs.value,
			id: vnode.attrs.id,
			required: vnode.attrs.required,
			placeholder: "password",
			autocomplete: vnode.attrs.autocomplete,
			pattern: "^.+$",
			minlength: 8,
			maxlength: 1000,
		}),
	]
}
