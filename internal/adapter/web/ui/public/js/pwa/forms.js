export const EmailInput = {
	view: vnode => [
		vnode.attrs.label ? m("label", {
			for: vnode.attrs.id,
			class: vnode.attrs.required ? "required" : null,
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
	view: vnode => [
		vnode.attrs.label ? m("label", {
			for: vnode.attrs.id,
			class: vnode.attrs.required ? "required" : null,
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

export const TOTPInput = {
	view: vnode => [
		vnode.attrs.label ? m("label", {
			for: vnode.attrs.id,
			class: vnode.attrs.required ? "required" : null,
		}, vnode.attrs.label) : null,
		m("input[type=text]", {
			oninput: vnode.attrs.oninput,
			value: vnode.attrs.value,
			id: vnode.attrs.id,
			required: vnode.attrs.required,
			placeholder: "123456",
			autocomplete: vnode.attrs.autocomplete,
			pattern: "^\\d+$",
			minlength: 6,
			maxlength: 6,
		}),
	]
}

export const RecoveryCodeInput = {
	view: vnode => [
		vnode.attrs.label ? m("label", {
			for: vnode.attrs.id,
			class: vnode.attrs.required ? "required" : null,
		}, vnode.attrs.label) : null,
		m("input[type=text]", {
			oninput: vnode.attrs.oninput,
			value: vnode.attrs.value,
			id: vnode.attrs.id,
			required: vnode.attrs.required,
			placeholder: "ABCDEFGHIJKLM",
			autocomplete: "off",
			pattern: "^[A-Z2-7]+$",
			minlength: 13,
			maxlength: 13,
		}),
	]
}
